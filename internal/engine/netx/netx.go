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
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/errorsx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/quicdialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Logger is the logger assumed by this package
type Logger interface {
	Debugf(format string, v ...interface{})
	Debug(message string)
}

// Dialer is the definition of dialer assumed by this package.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// QUICDialer is the definition of a dialer for QUIC assumed by this package.
type QUICDialer interface {
	DialContext(ctx context.Context, network, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error)
}

// TLSDialer is the definition of a TLS dialer assumed by this package.
type TLSDialer interface {
	DialTLSContext(ctx context.Context, network, address string) (net.Conn, error)
}

// HTTPRoundTripper is the definition of http.HTTPRoundTripper used by this package.
type HTTPRoundTripper interface {
	RoundTrip(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// Resolver is the interface we expect from a resolver
type Resolver interface {
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
	Network() string
	Address() string
}

// Config contains configuration for creating a new transport. When any
// field of Config is nil/empty, we will use a suitable default.
//
// We use different savers for different kind of events such that the
// user of this library can choose what to save.
type Config struct {
	BaseResolver        Resolver             // default: system resolver
	BogonIsError        bool                 // default: bogon is not error
	ByteCounter         *bytecounter.Counter // default: no explicit byte counting
	CacheResolutions    bool                 // default: no caching
	CertPool            *x509.CertPool       // default: use vendored gocertifi
	ContextByteCounting bool                 // default: no implicit byte counting
	DNSCache            map[string][]string  // default: cache is empty
	DialSaver           *trace.Saver         // default: not saving dials
	Dialer              Dialer               // default: dialer.DNSDialer
	FullResolver        Resolver             // default: base resolver + goodies
	QUICDialer          QUICDialer           // default: quicdialer.DNSDialer
	HTTP3Enabled        bool                 // default: disabled
	HTTPSaver           *trace.Saver         // default: not saving HTTP
	Logger              Logger               // default: no logging
	NoTLSVerify         bool                 // default: perform TLS verify
	ProxyURL            *url.URL             // default: no proxy
	ReadWriteSaver      *trace.Saver         // default: not saving read/write
	ResolveSaver        *trace.Saver         // default: not saving resolves
	TLSConfig           *tls.Config          // default: attempt using h2
	TLSDialer           TLSDialer            // default: dialer.TLSDialer
	TLSSaver            *trace.Saver         // default: not saving TLS
}

type tlsHandshaker interface {
	Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}

var defaultCertPool *x509.CertPool = netxlite.NewDefaultCertPool()

// NewResolver creates a new resolver from the specified config
func NewResolver(config Config) Resolver {
	if config.BaseResolver == nil {
		config.BaseResolver = &netxlite.ResolverSystem{}
	}
	var r Resolver = config.BaseResolver
	r = &resolver.AddressResolver{
		Resolver: netxlite.NewResolverLegacyAdapter(r),
	}
	if config.CacheResolutions {
		r = &resolver.CacheResolver{Resolver: r}
	}
	if config.DNSCache != nil {
		cache := &resolver.CacheResolver{Resolver: r, ReadOnly: true}
		for key, values := range config.DNSCache {
			cache.Set(key, values)
		}
		r = cache
	}
	if config.BogonIsError {
		r = resolver.BogonResolver{Resolver: r}
	}
	r = &errorsx.ErrorWrapperResolver{Resolver: r}
	if config.Logger != nil {
		r = &netxlite.ResolverLogger{
			Logger:   config.Logger,
			Resolver: netxlite.NewResolverLegacyAdapter(r),
		}
	}
	if config.ResolveSaver != nil {
		r = resolver.SaverResolver{Resolver: r, Saver: config.ResolveSaver}
	}
	return &resolver.IDNAResolver{Resolver: netxlite.NewResolverLegacyAdapter(r)}
}

// NewDialer creates a new Dialer from the specified config
func NewDialer(config Config) Dialer {
	if config.FullResolver == nil {
		config.FullResolver = NewResolver(config)
	}
	return dialer.New(&dialer.Config{
		ContextByteCounting: config.ContextByteCounting,
		DialSaver:           config.DialSaver,
		Logger:              config.Logger,
		ProxyURL:            config.ProxyURL,
		ReadWriteSaver:      config.ReadWriteSaver,
	}, config.FullResolver)
}

// NewQUICDialer creates a new DNS Dialer for QUIC, with the resolver from the specified config
func NewQUICDialer(config Config) QUICDialer {
	if config.FullResolver == nil {
		config.FullResolver = NewResolver(config)
	}
	var ql quicdialer.QUICListener = &netxlite.QUICListenerStdlib{}
	ql = &errorsx.ErrorWrapperQUICListener{QUICListener: ql}
	if config.ReadWriteSaver != nil {
		ql = &quicdialer.QUICListenerSaver{
			QUICListener: ql,
			Saver:        config.ReadWriteSaver,
		}
	}
	var d quicdialer.ContextDialer = &netxlite.QUICDialerQUICGo{
		QUICListener: ql,
	}
	d = &errorsx.ErrorWrapperQUICDialer{Dialer: d}
	if config.TLSSaver != nil {
		d = quicdialer.HandshakeSaver{Saver: config.TLSSaver, Dialer: d}
	}
	d = &netxlite.QUICDialerResolver{
		Resolver: netxlite.NewResolverLegacyAdapter(config.FullResolver),
		Dialer:   netxlite.NewQUICDialerFromContextDialerAdapter(d),
	}
	return d
}

// NewTLSDialer creates a new TLSDialer from the specified config
func NewTLSDialer(config Config) TLSDialer {
	if config.Dialer == nil {
		config.Dialer = NewDialer(config)
	}
	var h tlsHandshaker = &netxlite.TLSHandshakerConfigurable{}
	h = &errorsx.ErrorWrapperTLSHandshaker{TLSHandshaker: h}
	if config.Logger != nil {
		h = &netxlite.TLSHandshakerLogger{Logger: config.Logger, TLSHandshaker: h}
	}
	if config.TLSSaver != nil {
		h = tlsdialer.SaverTLSHandshaker{TLSHandshaker: h, Saver: config.TLSSaver}
	}
	if config.TLSConfig == nil {
		config.TLSConfig = &tls.Config{NextProtos: []string{"h2", "http/1.1"}}
	}
	if config.CertPool == nil {
		config.CertPool = defaultCertPool
	}
	config.TLSConfig.RootCAs = config.CertPool
	config.TLSConfig.InsecureSkipVerify = config.NoTLSVerify
	return &netxlite.TLSDialerLegacy{
		Config:        config.TLSConfig,
		Dialer:        netxlite.NewDialerLegacyAdapter(config.Dialer),
		TLSHandshaker: h,
	}
}

// NewHTTPTransport creates a new HTTPRoundTripper. You can further extend the returned
// HTTPRoundTripper before wrapping it into an http.Client.
func NewHTTPTransport(config Config) HTTPRoundTripper {
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
	txp := tInfo.Factory(httptransport.Config{
		Dialer: config.Dialer, QUICDialer: config.QUICDialer, TLSDialer: config.TLSDialer,
		TLSConfig: config.TLSConfig})
	transport := tInfo.TransportName

	if config.ByteCounter != nil {
		txp = httptransport.ByteCountingTransport{
			Counter: config.ByteCounter, RoundTripper: txp}
	}
	if config.Logger != nil {
		txp = &netxlite.HTTPTransportLogger{Logger: config.Logger, HTTPTransport: txp}
	}
	if config.HTTPSaver != nil {
		txp = httptransport.SaverMetadataHTTPTransport{
			RoundTripper: txp, Saver: config.HTTPSaver, Transport: transport}
		txp = httptransport.SaverBodyHTTPTransport{
			RoundTripper: txp, Saver: config.HTTPSaver}
		txp = httptransport.SaverPerformanceHTTPTransport{
			RoundTripper: txp, Saver: config.HTTPSaver}
		txp = httptransport.SaverTransactionHTTPTransport{
			RoundTripper: txp, Saver: config.HTTPSaver}
	}
	return txp
}

// httpTransportInfo contains the constructing function as well as the transport name
type httpTransportInfo struct {
	Factory       func(httptransport.Config) httptransport.RoundTripper
	TransportName string
}

var allTransportsInfo = map[bool]httpTransportInfo{
	false: {
		Factory:       httptransport.NewSystemTransport,
		TransportName: "tcp",
	},
	true: {
		Factory:       httptransport.NewHTTP3Transport,
		TransportName: "quic",
	},
}

// DNSClient is a DNS client. It wraps a Resolver and it possibly
// also wraps an HTTP client, but only when we're using DoH.
type DNSClient struct {
	Resolver
	httpClient *http.Client
}

// CloseIdleConnections closes idle connections, if any.
func (c DNSClient) CloseIdleConnections() {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
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
func NewDNSClient(config Config, URL string) (DNSClient, error) {
	return NewDNSClientWithOverrides(config, URL, "", "", "")
}

// NewDNSClientWithOverrides creates a new DNS client, similar to NewDNSClient,
// with the option to override the default Hostname and SNI.
func NewDNSClientWithOverrides(config Config, URL, hostOverride, SNIOverride,
	TLSVersion string) (DNSClient, error) {
	var c DNSClient
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
		return c, err
	}
	config.TLSConfig = &tls.Config{ServerName: SNIOverride}
	if err := netxlite.ConfigureTLSVersion(config.TLSConfig, TLSVersion); err != nil {
		return c, err
	}
	switch resolverURL.Scheme {
	case "system":
		c.Resolver = &netxlite.ResolverSystem{}
		return c, nil
	case "https":
		config.TLSConfig.NextProtos = []string{"h2", "http/1.1"}
		c.httpClient = &http.Client{Transport: NewHTTPTransport(config)}
		var txp resolver.RoundTripper = resolver.NewDNSOverHTTPSWithHostOverride(
			c.httpClient, URL, hostOverride)
		if config.ResolveSaver != nil {
			txp = resolver.SaverDNSTransport{
				RoundTripper: txp,
				Saver:        config.ResolveSaver,
			}
		}
		c.Resolver = resolver.NewSerialResolver(txp)
		return c, nil
	case "udp":
		dialer := NewDialer(config)
		endpoint, err := makeValidEndpoint(resolverURL)
		if err != nil {
			return c, err
		}
		var txp resolver.RoundTripper = resolver.NewDNSOverUDP(dialer, endpoint)
		if config.ResolveSaver != nil {
			txp = resolver.SaverDNSTransport{
				RoundTripper: txp,
				Saver:        config.ResolveSaver,
			}
		}
		c.Resolver = resolver.NewSerialResolver(txp)
		return c, nil
	case "dot":
		config.TLSConfig.NextProtos = []string{"dot"}
		tlsDialer := NewTLSDialer(config)
		endpoint, err := makeValidEndpoint(resolverURL)
		if err != nil {
			return c, err
		}
		var txp resolver.RoundTripper = resolver.NewDNSOverTLS(
			tlsDialer.DialTLSContext, endpoint)
		if config.ResolveSaver != nil {
			txp = resolver.SaverDNSTransport{
				RoundTripper: txp,
				Saver:        config.ResolveSaver,
			}
		}
		c.Resolver = resolver.NewSerialResolver(txp)
		return c, nil
	case "tcp":
		dialer := NewDialer(config)
		endpoint, err := makeValidEndpoint(resolverURL)
		if err != nil {
			return c, err
		}
		var txp resolver.RoundTripper = resolver.NewDNSOverTCP(
			dialer.DialContext, endpoint)
		if config.ResolveSaver != nil {
			txp = resolver.SaverDNSTransport{
				RoundTripper: txp,
				Saver:        config.ResolveSaver,
			}
		}
		c.Resolver = resolver.NewSerialResolver(txp)
		return c, nil
	default:
		return c, errors.New("unsupported resolver scheme")
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
