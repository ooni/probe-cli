package netx

//
// DNSTransport from Config.
//
// TODO(bassosimone): this code should be refactored to return
// a DNSTransport rather than a model.Resolver. With this in mind,
// I've named this file dnstransport.go.
//

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

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
