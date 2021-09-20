package measurex

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
)

// Measurer performs measurements.
//
// You call measurer methods to perform measurements. All methods
// will save measurements into the DB field as a side effect.
//
// Some methods will also return (a subset of) their measurement
// results when doing that is convenient.
//
// This implementation currently uses the Web Connectivity Test
// Helper (WCTH) to help with measuring HTTP endpoints. We'll use
// an ad-hoc, more effective test helper in the near future.
//
// Remarks
//
// Make sure to initialize all the fields marked as MANDATORY.
type Measurer struct {
	// DB is the MANDATORY database to use.
	DB DB

	// HTTPClient is the MANDATORY HTTP client for the WCTH.
	HTTPClient HTTPClient

	// Logger is the MANDATORY logger to use.
	Logger Logger

	// Origin is the MANDATORY measurements origin to use.
	Origin Origin

	// TLSHandshaker is the MANDATORY TLS handshaker.
	TLSHandshaker TLSHandshaker

	// WCTHURL is the MANDATORY URL of the WCTH.
	WCTHURL string
}

// NewMeasurement increments the DB's MeasurementID
// and returns such an ID for later usage.
//
// Every operation we perform (e.g., a TCP connect) saves
// measurements into mx.DB using separate tables.
//
// We save the MeasurementID for each operation.
//
// By calling NewMeasurement you increment such an ID
// which later allows you to separate measurements.
func (mx *Measurer) NewMeasurement() int64 {
	return mx.DB.NextMeasurement()
}

// LookupHostSystem performs a LookupHost using the system resolver.
//
// The system resolver is equivalent to calling getaddrinfo on Unix systems.
//
// Arguments
//
// - ctx is the context allowing to timeout the operation;
//
// - domain is the domain to lookup.
//
// Return value
//
// Either a list of resolved IP addresses or an error.
func (mx *Measurer) LookupHostSystem(
	ctx context.Context, domain string) (addrs []string, err error) {
	const timeout = 4 * time.Second
	mx.infof("LookupHost[getaddrinfo] %s (timeout %s)...", domain, timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	r := mx.newResolverSystem()
	defer r.CloseIdleConnections()
	return r.LookupHost(ctx, domain)
}

// newResolverSystem is a convenience factory for creating a
// system resolver that saves measurements into mx.DB.
func (mx *Measurer) newResolverSystem() Resolver {
	return WrapResolver(mx.Origin, mx.DB, netxlite.NewResolverStdlib(mx.Logger))
}

// newDialerWithSystemResolver is a convenience factory for creating
// a dialer that saves measurements into mx.DB.
func (mx *Measurer) newDialerWithSystemResolver() Dialer {
	r := mx.newResolverSystem()
	return WrapDialer(mx.Origin, mx.DB, netxlite.NewDialerWithResolver(
		mx.Logger, r,
	))
}

// netxliteDialerAdapter adapts measurex.Dialer to netxlite.Dialer.
type netxliteDialerAdapter struct {
	Dialer
}

// DialContext implements netxlite.Dialer.DialContext.
func (d *netxliteDialerAdapter) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	return d.Dialer.DialContext(ctx, network, address)
}

// newResolverUDP is a convenience factory for creating a resolver
// using UDP that saves measurements into mx.DB.
//
// Arguments
//
// - address is the resolver address (e.g., "1.1.1.1:53").
//
// Return value
//
// A Resolver.
func (mx *Measurer) newResolverUDP(address string) Resolver {
	// TODO(bassosimone): the resolver we compose here is missing
	// some capabilities like IDNA. We should instead have the proper
	// factory inside netxlite for creating this resolver.
	return WrapResolver(mx.Origin, mx.DB, &netxlite.ResolverLogger{
		Resolver: dnsx.NewSerialResolver(
			WrapDNSXRoundTripper(mx.DB, dnsx.NewDNSOverUDP(
				&netxliteDialerAdapter{mx.newDialerWithSystemResolver()},
				address,
			))),
		Logger: mx.Logger,
	})
}

// LookupHostUDP is like LookupHostSystem but uses an UDP resolver.
//
// Arguments
//
// - ctx is the context allowing to timeout the operation;
//
// - domain is the domain to resolve (e.g., "x.org");
//
// - address is the UDP resolver address (e.g., "dns.google:53").
//
// Return value
//
// Either the resolved addresses or an error.
func (mx *Measurer) LookupHostUDP(
	ctx context.Context, domain, address string) ([]string, error) {
	const timeout = 4 * time.Second
	mx.infof("LookupHost[udp://%s] %s (timeout %s)...", address, domain, timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	r := mx.newResolverUDP(address)
	defer r.CloseIdleConnections()
	return r.LookupHost(ctx, domain)
}

// LookupHTTPSSvcUDP issues an HTTPSSvc query for the given domain.
//
// Arguments
//
// - ctx is the context allowing to timeout the operation;
//
// - domain is the domain to resolve (e.g., "x.org");
//
// - address is the UDP resolver address (e.g., "dns.google:53").
//
// Return value
//
// Either the query result, on success, or an error.
func (mx *Measurer) LookupHTTPSSvcUDP(
	ctx context.Context, domain, address string) (HTTPSSvc, error) {
	const timeout = 4 * time.Second
	mx.infof("LookupHTTPSSvc[udp://%s] %s (timeout %s)...", address, domain, timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	r := mx.newResolverUDP(address)
	defer r.CloseIdleConnections()
	return r.LookupHTTPSSvcWithoutRetry(ctx, domain)
}

// newDialerWithSystemResolver is a convenience factory for creating
// a dialer that saves measurements into mx.DB.
func (mx *Measurer) newDialerWithoutResolver() Dialer {
	return WrapDialer(mx.Origin, mx.DB, netxlite.NewDialerWithoutResolver(
		mx.Logger,
	))
}

// TCPConnect establishes a connection with a TCP endpoint.
//
// Arguments
//
// - ctx is the context allowing to timeout the connect;
//
// - address is the TCP endpoint address (e.g., "8.8.4.4:443").
//
// Return value
//
// Either an established Conn or an error.
func (mx *Measurer) TCPConnect(ctx context.Context, address string) (Conn, error) {
	const timeout = 10 * time.Second
	mx.infof("TCPConnect %s (timeout %s)...", address, timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	d := mx.newDialerWithoutResolver()
	defer d.CloseIdleConnections()
	return d.DialContext(ctx, "tcp", address)
}

// TLSConnect connects and TLS handshakes with a TCP endpoint.
//
// Arguments
//
// - ctx is the context allowing to timeout the whole operation;
//
// - address is the endpoint address (e.g., "1.1.1.1:443");
//
// - config contains the TLS config (see below).
//
// TLS config
//
// You MUST set the following config fields:
//
// - ServerName to the desired SNI or InsecureSkipVerify to
// skip the certificate name verification;
//
// - RootCAs to nextlite.NewDefaultCertPool() output;
//
// - NextProtos to the desired ALPN ([]string{"h2", "http/1.1"} for
// HTTPS and []string{"dot"} for DNS-over-TLS).
//
// Caveats
//
// The mx.TLSHandshaker field could point to a TLS handshaker using
// the Go stdlib or one using gitlab.com/yawning/utls.git.
//
// In the latter case, the content of the ClientHello message
// will not only depend on the config field but also on the
// utls.ClientHelloID thay you're using.
//
// Return value
//
// Either an established TLSConn or an error.
func (mx *Measurer) TLSConnect(ctx context.Context,
	address string, config *tls.Config) (TLSConn, error) {
	conn, err := mx.TCPConnect(ctx, address)
	if err != nil {
		return nil, err
	}
	const timeout = 10 * time.Second
	mx.infof("TLSHandshake[SNI=%s,ALPN=%+v] %s (timeout %s)...",
		config.ServerName, config.NextProtos, address, timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return mx.TLSHandshaker.Handshake(ctx, conn, config)
}

// ErrUnknownHTTPEndpointNetwork indicates that we don't know
// how to handle the value of an HTTPEndpoint.Network.
var ErrUnknownHTTPEndpointNetwork = errors.New("unknown HTTPEndpoint.Network")

// HTTPEndpointGet performs a GET request for an HTTP endpoint.
//
// This function WILL NOT follow redirects. If there is a redirect
// you will see it inside the specific mx.DB table.
//
// Arguments
//
// - ctx is the context allowing to timeout the operation;
//
// - epnt is the HTTP endpoint.
//
// Return value
//
// Either an HTTP response, on success, or an error.
func (mx *Measurer) HTTPEndpointGet(
	ctx context.Context, epnt *HTTPEndpoint) (*http.Response, error) {
	switch epnt.Network {
	case NetworkQUIC:
		return nil, ErrUnknownHTTPEndpointNetwork
	case NetworkTCP:
		return mx.httpEndpointGetTCP(ctx, epnt)
	default:
		return nil, ErrUnknownHTTPEndpointNetwork
	}
}

// ErrUnknownHTTPEndpointURLScheme indicates that we don't know how to
// handle the value of an HTTPEndpoint.URLScheme.
var ErrUnknownHTTPEndpointURLScheme = errors.New("unknown HTTPEndpoint.URL.Scheme")

// httpEndpointGetTCP specializes HTTPSEndpointGet for HTTP and HTTPS.
func (mx *Measurer) httpEndpointGetTCP(
	ctx context.Context, epnt *HTTPEndpoint) (*http.Response, error) {
	switch epnt.URL.Scheme {
	case "http":
		return mx.httpEndpointGetHTTP(ctx, epnt)
	case "https":
		return mx.httpEndpointGetHTTPS(ctx, epnt)
	default:
		return nil, ErrUnknownHTTPEndpointURLScheme
	}
}

// httpEndpointGetHTTP specializes httpEndpointGetTCP for HTTP.
func (mx *Measurer) httpEndpointGetHTTP(
	ctx context.Context, epnt *HTTPEndpoint) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header
	conn, err := mx.TCPConnect(ctx, epnt.Address)
	if err != nil {
		return nil, err
	}
	defer conn.Close() // we own it
	clnt := NewHTTPClientWithoutRedirects(mx.Origin, mx.DB,
		NewHTTPTransportWithConn(mx.Origin, mx.Logger, mx.DB, conn))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt, req)
}

// httpEndpointGetHTTPS specializes httpEndpointGetTCP for HTTPS.
func (mx *Measurer) httpEndpointGetHTTPS(
	ctx context.Context, epnt *HTTPEndpoint) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header
	conn, err := mx.TLSConnect(ctx, epnt.Address, &tls.Config{
		ServerName: epnt.SNI,
		NextProtos: epnt.ALPN,
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close() // we own it
	clnt := NewHTTPClientWithoutRedirects(mx.Origin, mx.DB,
		NewHTTPTransportWithTLSConn(mx.Origin, mx.Logger, mx.DB, conn))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt, req)
}

func (mx *Measurer) httpClientDo(ctx context.Context, clnt HTTPClient,
	epnt *HTTPEndpoint, req *http.Request) (*http.Response, error) {
	const timeout = 15 * time.Second
	mx.infof("HTTPGet[epnt=%s] %s (timeout %s)...",
		epnt.Address, epnt.URL.String(), timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return clnt.Do(req.WithContext(ctx))
}

// EndpointNetwork is the network of an endpoint.
type EndpointNetwork string

const (
	// NetworkTCP identifies endpoints using TCP.
	NetworkTCP = EndpointNetwork("tcp")

	// NetworkQUIC identifies endpoints using QUIC.
	NetworkQUIC = EndpointNetwork("quic")
)

// Endpoint is an endpoint for a domain.
type Endpoint struct {
	// Network is the network (e.g., "tcp", "quic")
	Network EndpointNetwork

	// Address is the endpoint address (e.g., "8.8.8.8:443")
	Address string
}

// String converts an endpoint to a string (e.g., "8.8.8.8:443/tcp")
func (e *Endpoint) String() string {
	return fmt.Sprintf("%s/%s", e.Address, e.Network)
}

// ErrLookupEndpoints failed indicates that we could not
// successfully lookup the endpoints for a domain.
var ErrLookupEndpoints = errors.New("endpoints lookup failed")

// LookupEndpoints discovers the endpoints for a domain.
//
// This function performs two lookups:
//
// - with the system resolver;
//
// - with a DNS over UDP resolver.
//
// Arguments
//
// - ctx is the context carrying timeouts;
//
// - domain is the domain to lookup endpoints for;
//
// - port is the port we want to use;
//
// - address is the address of a DNS over UDP resolver.
//
// Return value
//
// Returns either a list of endpoints or an error. The error will just
// indicate that we could not resolve _any_ endpoint. Precise results
// regarding each performed operation are into the mx.DB field.
func (mx *Measurer) LookupEndpoints(
	ctx context.Context, domain, port, address string) ([]*Endpoint, error) {
	udpAddrs, _ := mx.LookupHostUDP(ctx, domain, address)
	systemAddrs, _ := mx.LookupHostSystem(ctx, domain)
	var out []*Endpoint
	out = append(out, mx.parseLookupHostReply(port, systemAddrs)...)
	out = append(out, mx.parseLookupHostReply(port, udpAddrs)...)
	out = mx.mergeEndpoints(out)
	if len(out) < 1 {
		return nil, ErrLookupEndpoints
	}
	return out, nil
}

// mergeEndpoints merges duplicate endpoints in the input list.
//
// Arguments
//
// - input is the input list of endpoints to merge.
//
// Return value
//
// A list where duplicates have been removed.
func (mx *Measurer) mergeEndpoints(input []*Endpoint) (out []*Endpoint) {
	var (
		tcp  = make(map[string]int)
		quic = make(map[string]int)
	)
	for _, epnt := range input {
		switch epnt.Network {
		case NetworkQUIC:
			quic[epnt.Address]++
		case NetworkTCP:
			tcp[epnt.Address]++
		}
	}
	for addr := range tcp {
		out = append(out, &Endpoint{
			Network: NetworkTCP,
			Address: addr,
		})
	}
	for addr := range quic {
		out = append(out, &Endpoint{
			Network: NetworkQUIC,
			Address: addr,
		})
	}
	return
}

// ErrCannotDeterminePortFromURL indicates that we could not determine
// the correct port from the URL authority and scheme.
var ErrCannotDeterminePortFromURL = errors.New("cannot determine port from URL")

// urlPort returns the port implied by an URL.
//
// If the URL contains an explicit port, we return it. Otherwise we
// attempt to guess the port based on the URL scheme.
//
// We currently recognize only these schemes:
//
// - "https";
//
// - "http".
//
// Arguments
//
// - URL is the URL for which to guess the port.
//
// Return value
//
// Either a string containing the port or an error.
func (mx *Measurer) urlPort(URL *url.URL) (string, error) {
	switch {
	case URL.Port() != "":
		return URL.Port(), nil
	case URL.Scheme == "https":
		return "443", nil
	case URL.Scheme == "http":
		return "80", nil
	default:
		return "", ErrCannotDeterminePortFromURL
	}
}

// HTTPEndpoint is an HTTP/HTTPS/HTTP3 endpoint.
type HTTPEndpoint struct {
	// Domain is the endpoint domain (e.g., "dns.google").
	Domain string

	// Network is the network (e.g., "tcp" or "quic").
	Network EndpointNetwork

	// Address is the endpoint address (e.g., "8.8.8.8:443").
	Address string

	// SNI is the SNI to use (only used with URL.scheme == "https").
	SNI string

	// ALPN is the ALPN to use (only used with URL.scheme == "https").
	ALPN []string

	// URL is the endpoint URL.
	URL *url.URL

	// Header contains request headers.
	Header http.Header
}

// String converts an HTTP endpoint to a string (e.g., "8.8.8.8:443/tcp")
func (e *HTTPEndpoint) String() string {
	return fmt.Sprintf("%s/%s", e.Address, e.Network)
}

// LookupHTTPEndpoints is like LookupEndpoints but performs a
// specialized lookup for an HTTP/HTTPS URL. Such a lookup also
// includes querying the WCTH to discover extra endpoints.
//
// Arguments
//
// - ctx is the context carrying timeouts;
//
// - URL is the URL to perform the lookup for;
//
// - address is the address of the DNS over
// UDP server to use.
//
// Return value
//
// Returns either a list of endpoints or an error. The returned error
// only indicates we could not fetch _any_ endpoint. Check into the
// database (i.e., mx.DB) for precise results of each operation.
func (mx *Measurer) LookupHTTPEndpoints(
	ctx context.Context, URL *url.URL, address string) ([]*HTTPEndpoint, error) {
	port, err := mx.urlPort(URL)
	if err != nil {
		return nil, err
	}
	httpsSvcInfo, _ := mx.LookupHTTPSSvcUDP(ctx, URL.Hostname(), address)
	endpoints, _ := mx.LookupEndpoints(ctx, URL.Hostname(), port, address)
	endpoints = append(endpoints, mx.parseHTTPSSvcReply(port, httpsSvcInfo)...)
	endpoints, _ = mx.lookupWCTH(ctx, URL, endpoints, port)
	endpoints = mx.mergeEndpoints(endpoints)
	if len(endpoints) < 1 {
		return nil, ErrLookupEndpoints
	}
	return mx.newHTTPEndpoints(URL, endpoints), nil
}

// newHTTPEndpoints takes in input a list of Endpoint and
// returns in output a list of HTTPEndpoint.
//
// Arguments
//
// - URL is the URL for which we're discovering HTTPEndpoint;
//
// - endpoints is the list of discovered Endpoint.
//
// Return value
//
// The list of HTTPEndpoint.
func (mx *Measurer) newHTTPEndpoints(
	URL *url.URL, endpoints []*Endpoint) (out []*HTTPEndpoint) {
	for _, epnt := range endpoints {
		out = append(out, &HTTPEndpoint{
			Domain:  URL.Hostname(),
			Network: epnt.Network,
			Address: epnt.Address,
			SNI:     URL.Hostname(),
			ALPN:    mx.alpnForHTTPEndpoint(epnt.Network),
			URL:     URL,
			Header:  NewHTTPRequestHeaderForMeasuring(),
		})
	}
	return
}

// alpnForHTTPEndpoint takes in input the network of an endpoint
// (i.e., "tcp" or "quic") and returns the corresponding ALPN.
//
// Arguments
//
// - network is the network of the endpoint.
//
// Return value
//
// The corresponding ALPN. If we do not recognize the input
// network we return a nil string array.
func (mx *Measurer) alpnForHTTPEndpoint(network EndpointNetwork) []string {
	switch network {
	case NetworkQUIC:
		return []string{"h3"}
	case NetworkTCP:
		return []string{"h2", "http/1.1"}
	default:
		return nil
	}
}

// lookupWCTH performs an Endpoint looking using the WCTH (i.e.,
// the Web Connectivity Test Helper) web service.
//
// Arguments
//
// - ctx is the context carrying timeouts;
//
// - URL is the URL for which we're looking up endpoints;
//
// - endpoints is the list of endpoints discovered so far using
// the means available to the probe (e.g., DNS);
//
// - port is the port for the endpoints.
//
// Return value
//
// Either a list of endpoints (which may possibly be empty) in case
// of success or an error in case of failure.
func (mx *Measurer) lookupWCTH(ctx context.Context,
	URL *url.URL, endpoints []*Endpoint, port string) ([]*Endpoint, error) {
	const timeout = 30 * time.Second
	mx.infof("WCTH[backend=%s] %s %+v %s (timeout %s)...",
		mx.WCTHURL, URL.String(), endpoints, port, timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	w := NewWCTHWorker(mx.Logger, mx.DB, mx.HTTPClient, mx.WCTHURL)
	resp, err := w.Run(ctx, URL, mx.onlyTCPEndpoints(endpoints))
	if err != nil {
		return nil, err
	}
	for _, addr := range resp.DNS.Addrs {
		addrport := net.JoinHostPort(addr, port)
		endpoints = append(endpoints, &Endpoint{
			Network: NetworkTCP,
			Address: addrport,
		})
	}
	return endpoints, nil
}

// onlyTCPEndpoints takes in input a list of endpoints and returns
// in output a list of endpoints only containing the TCP ones.
func (mx *Measurer) onlyTCPEndpoints(endpoints []*Endpoint) (out []string) {
	for _, epnt := range endpoints {
		switch epnt.Network {
		case NetworkTCP:
			out = append(out, epnt.Address)
		}
	}
	return
}

// parseLookupHostReply builds a list of endpoints from a LookupHost reply.
//
// Arguments:
//
// - port is the port to use for the endpoints;
//
// - addrs is the possibly empty list of addresses from LookupHost.
//
// Return value
//
// A possibly empty list of endpoints.
func (mx *Measurer) parseLookupHostReply(port string, addrs []string) (out []*Endpoint) {
	for _, addr := range addrs {
		out = append(out, &Endpoint{
			Network: "tcp",
			Address: net.JoinHostPort(addr, port),
		})
	}
	return
}

// ParseHTTPSSvcReply builds a list of endpoints from the LookupHTTPSSvc result.
//
// Arguments
//
// - port is the port for the endpoints;
//
// - info is either nil or contains the result of the LookupHostHTTPSSvc call.
//
// Return value
//
// A possibly-empty list of endpoints.
func (mx *Measurer) parseHTTPSSvcReply(port string, info HTTPSSvc) (out []*Endpoint) {
	if info == nil {
		return
	}
	for _, proto := range info.ALPN() {
		switch proto {
		case "h3": // we do not support experimental protocols like h3-29 anymore
			for _, addr := range info.IPv4Hint() {
				out = append(out, &Endpoint{
					Network: "quic",
					Address: net.JoinHostPort(addr, port),
				})
			}
			for _, addr := range info.IPv6Hint() {
				out = append(out, &Endpoint{
					Network: "quic",
					Address: net.JoinHostPort(addr, port),
				})
			}
			return // we found what we were looking for
		}
	}
	return
}

// infof formats and logs an informational message using mx.Logger.
func (mx *Measurer) infof(format string, v ...interface{}) {
	mx.Logger.Infof(format, v...)
}
