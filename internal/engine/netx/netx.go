// Package netx contains code to perform network measurements.
//
// This library contains replacements for commonly used standard library
// interfaces that facilitate seamless network measurements. By using
// such replacements, as opposed to standard library interfaces, we can:
//
// * save the timing of HTTP events (e.g. received response headers)
// * save the timing and result of every Connect, Read, Write, Close operation
// * save the timing and result of the TLS handshake (including certificates)
//
// By default, this library uses the system resolver. In addition, it
// is possible to configure alternative DNS transports and remote
// servers. We support DNS over UDP, DNS over TCP, DNS over TLS (DoT),
// and DNS over HTTPS (DoH). When using an alternative transport, we
// are also able to intercept and save DNS messages, as well as any
// other interaction with the remote server (e.g., the result of the
// TLS handshake for DoT and DoH).
//
// We described the design and implementation of the most recent version of
// this package at <https://github.com/ooni/probe-engine/issues/359>. Such
// issue also links to a previous design document.
package netx

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// Config contains configuration for creating a new transport. When any
// field of Config is nil/empty, we will use a suitable default.
//
// We use different savers for different kind of events such that the
// user of this library can choose what to save.
type Config struct {
	BaseResolver        model.Resolver       // default: system resolver
	BogonIsError        bool                 // default: bogon is not error
	ByteCounter         *bytecounter.Counter // default: no explicit byte counting
	CacheResolutions    bool                 // default: no caching
	ContextByteCounting bool                 // default: no implicit byte counting
	DNSCache            map[string][]string  // default: cache is empty
	Dialer              model.Dialer         // default: dialer.DNSDialer
	FullResolver        model.Resolver       // default: base resolver + goodies
	QUICDialer          model.QUICDialer     // default: quicdialer.DNSDialer
	HTTP3Enabled        bool                 // default: disabled
	Logger              model.Logger         // default: no logging
	ProxyURL            *url.URL             // default: no proxy
	ReadWriteSaver      *tracex.Saver        // default: not saving I/O events
	Saver               *tracex.Saver        // default: not saving non-I/O events
	TLSConfig           *tls.Config          // default: attempt using h2
	TLSDialer           model.TLSDialer      // default: dialer.TLSDialer
}

// NewResolver creates a new resolver from the specified config
func NewResolver(config Config) model.Resolver {
	if config.BaseResolver == nil {
		config.BaseResolver = netxlite.NewResolverSystem()
	}
	r := netxlite.WrapResolver(
		model.ValidLoggerOrDefault(config.Logger),
		config.BaseResolver,
	)
	r = MaybeWrapWithCachingResolver(config.CacheResolutions, r)
	r = MaybeWrapWithStaticDNSCache(config.DNSCache, r)
	r = netxlite.MaybeWrapWithBogonResolver(config.BogonIsError, r)
	return config.Saver.WrapResolver(r) // WAI when config.Saver==nil
}

// NewDialer creates a new Dialer from the specified config
func NewDialer(config Config) model.Dialer {
	if config.FullResolver == nil {
		config.FullResolver = NewResolver(config)
	}
	logger := model.ValidLoggerOrDefault(config.Logger)
	d := netxlite.NewDialerWithResolver(
		logger, config.FullResolver, config.Saver.NewConnectObserver(),
		config.ReadWriteSaver.NewReadWriteObserver(),
	)
	d = netxlite.NewMaybeProxyDialer(d, config.ProxyURL)
	d = bytecounter.MaybeWrapWithContextAwareDialer(config.ContextByteCounting, d)
	return d
}

// NewQUICDialer creates a new DNS Dialer for QUIC, with the resolver from the specified config
func NewQUICDialer(config Config) model.QUICDialer {
	if config.FullResolver == nil {
		config.FullResolver = NewResolver(config)
	}
	// TODO(bassosimone): we should count the bytes consumed by this QUIC dialer
	ql := config.ReadWriteSaver.WrapQUICListener(netxlite.NewQUICListener())
	logger := model.ValidLoggerOrDefault(config.Logger)
	return netxlite.NewQUICDialerWithResolver(ql, logger, config.FullResolver, config.Saver)
}

// NewTLSDialer creates a new TLSDialer from the specified config
func NewTLSDialer(config Config) model.TLSDialer {
	if config.Dialer == nil {
		config.Dialer = NewDialer(config)
	}
	logger := model.ValidLoggerOrDefault(config.Logger)
	thx := netxlite.NewTLSHandshakerStdlib(logger)
	thx = config.Saver.WrapTLSHandshaker(thx) // WAI even when config.Saver is nil
	tlsConfig := netxlite.ClonedTLSConfigOrNewEmptyConfig(config.TLSConfig)
	return netxlite.NewTLSDialerWithConfig(config.Dialer, thx, tlsConfig)
}

// NewHTTPTransport creates a new HTTPRoundTripper. You can further extend the returned
// HTTPRoundTripper before wrapping it into an http.Client.
func NewHTTPTransport(config Config) model.HTTPTransport {
	if config.Dialer == nil {
		config.Dialer = NewDialer(config)
	}
	if config.TLSDialer == nil {
		config.TLSDialer = NewTLSDialer(config)
	}
	if config.QUICDialer == nil {
		config.QUICDialer = NewQUICDialer(config)
	}
	tInfo := allTransportsInfo[config.HTTP3Enabled]
	txp := tInfo.Factory(httpTransportConfig{
		Dialer:     config.Dialer,
		Logger:     model.ValidLoggerOrDefault(config.Logger),
		QUICDialer: config.QUICDialer,
		TLSDialer:  config.TLSDialer,
		TLSConfig:  config.TLSConfig,
	})
	// TODO(bassosimone): I am not super convinced by this code because it
	// seems we're currently counting bytes twice in some cases. I think we
	// should review how we're counting bytes and using netx currently.
	txp = config.ByteCounter.MaybeWrapHTTPTransport(txp)                 // WAI with ByteCounter == nil
	const defaultSnapshotSize = 0                                        // means: use the default snapsize
	return config.Saver.MaybeWrapHTTPTransport(txp, defaultSnapshotSize) // WAI with Saver == nil
}

// httpTransportInfo contains the constructing function as well as the transport name
type httpTransportInfo struct {
	Factory       func(httpTransportConfig) model.HTTPTransport
	TransportName string
}

var allTransportsInfo = map[bool]httpTransportInfo{
	false: {
		Factory:       newSystemTransport,
		TransportName: "tcp",
	},
	true: {
		Factory:       newHTTP3Transport,
		TransportName: "quic",
	},
}

// NewDNSClient creates a new DNS client. The config argument is used to
// create the underlying Dialer and/or HTTP transport, if needed. The URL
// argument describes the kind of client that we want to make:
//
// - if the URL is `doh://powerdns`, `doh://google` or `doh://cloudflare` or the URL
// starts with `https://`, then we create a DoH client.
//
// - if the URL is `` or `system:///`, then we create a system client,
// i.e. a client using the system resolver.
//
// - if the URL starts with `udp://`, then we create a client using
// a resolver that uses the specified UDP endpoint.
//
// We return error if the URL does not parse or the URL scheme does not
// fall into one of the cases described above.
//
// If config.ResolveSaver is not nil and we're creating an underlying
// resolver where this is possible, we will also save events.
func NewDNSClient(config Config, URL string) (model.Resolver, error) {
	return NewDNSClientWithOverrides(config, URL, "", "", "")
}

// NewDNSClientWithOverrides creates a new DNS client, similar to NewDNSClient,
// with the option to override the default Hostname and SNI.
func NewDNSClientWithOverrides(config Config, URL, hostOverride, SNIOverride,
	TLSVersion string) (model.Resolver, error) {
	switch URL {
	case "doh://powerdns":
		URL = "https://doh.powerdns.org/"
	case "doh://google":
		URL = "https://dns.google/dns-query"
	case "doh://cloudflare":
		URL = "https://cloudflare-dns.com/dns-query"
	case "":
		URL = "system:///"
	}
	resolverURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	config.TLSConfig = &tls.Config{ServerName: SNIOverride}
	if err := netxlite.ConfigureTLSVersion(config.TLSConfig, TLSVersion); err != nil {
		return nil, err
	}
	switch resolverURL.Scheme {
	case "system":
		return netxlite.NewResolverSystem(), nil
	case "https":
		config.TLSConfig.NextProtos = []string{"h2", "http/1.1"}
		httpClient := &http.Client{Transport: NewHTTPTransport(config)}
		var txp model.DNSTransport = netxlite.NewUnwrappedDNSOverHTTPSTransportWithHostOverride(
			httpClient, URL, hostOverride)
		txp = config.Saver.WrapDNSTransport(txp) // safe when config.Saver == nil
		return netxlite.NewUnwrappedSerialResolver(txp), nil
	case "udp":
		dialer := NewDialer(config)
		endpoint, err := makeValidEndpoint(resolverURL)
		if err != nil {
			return nil, err
		}
		var txp model.DNSTransport = netxlite.NewUnwrappedDNSOverUDPTransport(
			dialer, endpoint)
		txp = config.Saver.WrapDNSTransport(txp) // safe when config.Saver == nil
		return netxlite.NewUnwrappedSerialResolver(txp), nil
	case "dot":
		config.TLSConfig.NextProtos = []string{"dot"}
		tlsDialer := NewTLSDialer(config)
		endpoint, err := makeValidEndpoint(resolverURL)
		if err != nil {
			return nil, err
		}
		var txp model.DNSTransport = netxlite.NewUnwrappedDNSOverTLSTransport(
			tlsDialer.DialTLSContext, endpoint)
		txp = config.Saver.WrapDNSTransport(txp) // safe when config.Saver == nil
		return netxlite.NewUnwrappedSerialResolver(txp), nil
	case "tcp":
		dialer := NewDialer(config)
		endpoint, err := makeValidEndpoint(resolverURL)
		if err != nil {
			return nil, err
		}
		var txp model.DNSTransport = netxlite.NewUnwrappedDNSOverTCPTransport(
			dialer.DialContext, endpoint)
		txp = config.Saver.WrapDNSTransport(txp) // safe when config.Saver == nil
		return netxlite.NewUnwrappedSerialResolver(txp), nil
	default:
		return nil, errors.New("unsupported resolver scheme")
	}
}

// makeValidEndpoint makes a valid endpoint for DoT and Do53 given the
// input URL representing such endpoint. Specifically, we are
// concerned with the case where the port is missing. In such a
// case, we ensure that we are using the default port 853 for DoT
// and default port 53 for TCP and UDP.
func makeValidEndpoint(URL *url.URL) (string, error) {
	// Implementation note: when we're using a quoted IPv6
	// address, URL.Host contains the quotes but instead the
	// return value from URL.Hostname() does not.
	//
	// For example:
	//
	// - Host: [2620:fe::9]
	// - Hostname(): 2620:fe::9
	//
	// We need to keep this in mind when trying to determine
	// whether there is also a port or not.
	//
	// So the first step is to check whether URL.Host is already
	// a whatever valid TCP/UDP endpoint and, if so, use it.
	if _, _, err := net.SplitHostPort(URL.Host); err == nil {
		return URL.Host, nil
	}
	// The second step is to assume that appending the default port
	// to a host parsed by url.Parse should be giving us a valid
	// endpoint. The possibilities in fact are:
	//
	// 1. domain w/o port
	// 2. IPv4 w/o port
	// 3. square bracket quoted IPv6 w/o port
	// 4. other
	//
	// In the first three cases, appending a port leads us to a
	// good endpoint. The fourth case does not.
	//
	// For this reason we check again whether we can split it using
	// net.SplitHostPort. If we cannot, we were in case four.
	host := URL.Host
	if URL.Scheme == "dot" {
		host += ":853"
	} else {
		host += ":53"
	}
	if _, _, err := net.SplitHostPort(host); err != nil {
		return "", err
	}
	// Otherwise it's one of the three valid cases above.
	return host, nil
}
