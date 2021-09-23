package measurex

//
// Measurer
//
// High-level API for running measurements. The code in here
// has been designed to easily implement the new websteps
// network experiment, which is quite complex. It should be
// possible to write most other experiments using a Measurer.
//

import (
	"context"
	"crypto/tls"
	"errors"
	stdlog "log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Measurer performs measurements. If you don't use a factory
// for creating this type, make sure you set all the MANDATORY fields.
type Measurer struct {
	// Begin is when we started measuring (this field is MANDATORY).
	Begin time.Time

	// HTTPClient is the MANDATORY HTTP client for the WCTH.
	HTTPClient HTTPClient

	// Logger is the MANDATORY logger to use.
	Logger Logger

	// MeasureURLHelper is the OPTIONAL test helper to use when
	// we're measuring using the MeasureURL function. If this field
	// is not set, we'll not be using any helper.
	MeasureURLHelper MeasureURLHelper

	// Resolvers is the MANDATORY list of resolvers.
	Resolvers []*ResolverInfo

	// TLSHandshaker is the MANDATORY TLS handshaker.
	TLSHandshaker netxlite.TLSHandshaker
}

// NewMeasurerWithDefaultSettings creates a new Measurer
// instance using the most default settings.
func NewMeasurerWithDefaultSettings() *Measurer {
	return &Measurer{
		Begin:      time.Now(),
		HTTPClient: &http.Client{},
		Logger:     log.Log,
		Resolvers: []*ResolverInfo{{
			Network: "system",
			Address: "",
		}, {
			Network: "udp",
			Address: "8.8.4.4:53",
		}},
		TLSHandshaker: netxlite.NewTLSHandshakerStdlib(log.Log),
	}
}

// LookupHostSystem performs a LookupHost using the system resolver.
func (mx *Measurer) LookupHostSystem(ctx context.Context, domain string) *DNSMeasurement {
	const timeout = 4 * time.Second
	ol := newOperationLogger(mx.Logger, "LookupHost %s with getaddrinfo", domain)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	r := mx.NewResolverSystem(db, mx.Logger)
	defer r.CloseIdleConnections()
	_, err := r.LookupHost(ctx, domain)
	ol.Stop(err)
	return &DNSMeasurement{
		Domain:      domain,
		Measurement: db.AsMeasurement(),
	}
}

// lookupHostForeign performs a LookupHost using a "foreign" resolver.
func (mx *Measurer) lookupHostForeign(
	ctx context.Context, domain string, r Resolver) *DNSMeasurement {
	const timeout = 4 * time.Second
	ol := newOperationLogger(mx.Logger, "LookupHost %s with %s", domain, r.Network())
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	_, err := mx.WrapResolver(db, r).LookupHost(ctx, domain)
	ol.Stop(err)
	return &DNSMeasurement{
		Domain:      domain,
		Measurement: db.AsMeasurement(),
	}
}

// LookupHostUDP is like LookupHostSystem but uses an UDP resolver.
//
// Arguments:
//
// - ctx is the context allowing to timeout the operation;
//
// - domain is the domain to resolve (e.g., "x.org");
//
// - address is the UDP resolver address (e.g., "dns.google:53").
//
// Returns a DNSMeasurement.
func (mx *Measurer) LookupHostUDP(
	ctx context.Context, domain, address string) *DNSMeasurement {
	const timeout = 4 * time.Second
	ol := newOperationLogger(mx.Logger, "LookupHost %s with %s/udp", domain, address)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	r := mx.NewResolverUDP(db, mx.Logger, address)
	defer r.CloseIdleConnections()
	_, err := r.LookupHost(ctx, domain)
	ol.Stop(err)
	return &DNSMeasurement{
		Domain:      domain,
		Measurement: db.AsMeasurement(),
	}
}

// LookupHTTPSSvcUDP issues an HTTPSSvc query for the given domain.
//
// Arguments:
//
// - ctx is the context allowing to timeout the operation;
//
// - domain is the domain to resolve (e.g., "x.org");
//
// - address is the UDP resolver address (e.g., "dns.google:53").
//
// Returns a DNSMeasurement.
func (mx *Measurer) LookupHTTPSSvcUDP(
	ctx context.Context, domain, address string) *DNSMeasurement {
	const timeout = 4 * time.Second
	ol := newOperationLogger(mx.Logger, "LookupHTTPSvc %s with %s/udp", domain, address)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	r := mx.NewResolverUDP(db, mx.Logger, address)
	defer r.CloseIdleConnections()
	_, err := r.LookupHTTPSSvcWithoutRetry(ctx, domain)
	ol.Stop(err)
	return &DNSMeasurement{
		Domain:      domain,
		Measurement: db.AsMeasurement(),
	}
}

// lookupHTTPSSvcUDPForeign is like LookupHTTPSSvcUDP
// except that it uses a "foreign" resolver.
func (mx *Measurer) lookupHTTPSSvcUDPForeign(
	ctx context.Context, domain string, r Resolver) *DNSMeasurement {
	const timeout = 4 * time.Second
	ol := newOperationLogger(mx.Logger, "LookupHTTPSvc %s with %s", domain, r.Address())
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	_, err := mx.WrapResolver(db, r).LookupHTTPSSvcWithoutRetry(ctx, domain)
	ol.Stop(err)
	return &DNSMeasurement{
		Domain:      domain,
		Measurement: db.AsMeasurement(),
	}
}

// TCPConnect establishes a connection with a TCP endpoint.
//
// Arguments:
//
// - ctx is the context allowing to timeout the connect;
//
// - address is the TCP endpoint address (e.g., "8.8.4.4:443").
//
// Returns an EndpointMeasurement.
func (mx *Measurer) TCPConnect(ctx context.Context, address string) *EndpointMeasurement {
	db := &MeasurementDB{}
	conn, _ := mx.tcpConnect(ctx, db, address)
	measurement := db.AsMeasurement()
	if conn != nil {
		conn.Close()
	}
	return &EndpointMeasurement{
		Endpoint: (&Endpoint{
			Network: NetworkTCP,
			Address: address,
		}).String(),
		Measurement: measurement,
	}
}

// tcpConnect is like TCPConnect but does not create a new measurement.
func (mx *Measurer) tcpConnect(ctx context.Context, db WritableDB, address string) (Conn, error) {
	const timeout = 10 * time.Second
	ol := newOperationLogger(mx.Logger, "TCPConnect %s", address)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	d := mx.NewDialerWithoutResolver(db, mx.Logger)
	defer d.CloseIdleConnections()
	conn, err := d.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	return conn, err
}

// TLSConnectAndHandshake connects and TLS handshakes with a TCP endpoint.
//
// Arguments:
//
// - ctx is the context allowing to timeout the whole operation;
//
// - address is the endpoint address (e.g., "1.1.1.1:443");
//
// - config contains the TLS config (see below).
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
// Caveats:
//
// The mx.TLSHandshaker field could point to a TLS handshaker using
// the Go stdlib or one using gitlab.com/yawning/utls.git.
//
// In the latter case, the content of the ClientHello message
// will not only depend on the config field but also on the
// utls.ClientHelloID thay you're using.
//
// Returns an EndpointMeasurement.
func (mx *Measurer) TLSConnectAndHandshake(ctx context.Context,
	address string, config *tls.Config) *EndpointMeasurement {
	db := &MeasurementDB{}
	conn, _ := mx.tlsConnectAndHandshake(ctx, db, address, config)
	measurement := db.AsMeasurement()
	if conn != nil {
		conn.Close()
	}
	return &EndpointMeasurement{
		Endpoint: (&Endpoint{
			Network: NetworkTCP,
			Address: address,
		}).String(),
		Measurement: measurement,
	}
}

// tlsConnectAndHandshake is like TLSConnectAndHandshake
// but does not create a new measurement.
func (mx *Measurer) tlsConnectAndHandshake(ctx context.Context,
	db WritableDB, address string, config *tls.Config) (netxlite.TLSConn, error) {
	conn, err := mx.tcpConnect(ctx, db, address)
	if err != nil {
		return nil, err
	}
	const timeout = 10 * time.Second
	ol := newOperationLogger(mx.Logger,
		"TLSHandshake %s with sni=%s", address, config.ServerName)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	th := mx.WrapTLSHandshaker(db, mx.TLSHandshaker)
	tlsConn, _, err := th.Handshake(ctx, conn, config)
	ol.Stop(err)
	// cast safe according to the docs of netxlite's handshaker
	return tlsConn.(netxlite.TLSConn), err
}

// QUICHandshake connects and TLS handshakes with a QUIC endpoint.
//
// Arguments:
//
// - ctx is the context allowing to timeout the whole operation;
//
// - address is the endpoint address (e.g., "1.1.1.1:443");
//
// - config contains the TLS config (see below).
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
// Returns an EndpointMeasurement.
func (mx *Measurer) QUICHandshake(ctx context.Context, address string,
	config *tls.Config) *EndpointMeasurement {
	db := &MeasurementDB{}
	sess, _ := mx.quicHandshake(ctx, db, address, config)
	measurement := db.AsMeasurement()
	if sess != nil {
		// TODO(bassosimone): close session with correct message
		sess.CloseWithError(0, "")
	}
	return &EndpointMeasurement{
		Endpoint: (&Endpoint{
			Network: NetworkQUIC,
			Address: address,
		}).String(),
		Measurement: measurement,
	}
}

// quicHandshake is like QUICHandshake but does not create a new measurement.
func (mx *Measurer) quicHandshake(ctx context.Context, db WritableDB,
	address string, config *tls.Config) (quic.EarlySession, error) {
	const timeout = 10 * time.Second
	ol := newOperationLogger(mx.Logger,
		"QUICHandshake %s with sni=%s", address, config.ServerName)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	qd := mx.NewQUICDialerWithoutResolver(db, mx.Logger)
	defer qd.CloseIdleConnections()
	sess, err := qd.DialContext(ctx, "udp", address, config, &quic.Config{})
	ol.Stop(err)
	return sess, err
}

// HTTPEndpointGet performs a GET request for an HTTP endpoint.
//
// This function WILL NOT follow redirects. If there is a redirect
// you will see it inside the specific database table.
//
// Arguments:
//
// - ctx is the context allowing to timeout the operation;
//
// - epnt is the HTTP endpoint;
//
// - jar is the cookie jar to use.
//
// Returns a measurement. The returned measurement is empty if
// the endpoint is misconfigured or the URL has an unknown scheme.
func (mx *Measurer) HTTPEndpointGet(
	ctx context.Context, epnt *HTTPEndpoint, jar http.CookieJar) *HTTPEndpointMeasurement {
	resp, m, _ := mx.httpEndpointGet(ctx, epnt, jar)
	if resp != nil {
		resp.Body.Close()
	}
	return m
}

var (
	errUnknownHTTPEndpointURLScheme = errors.New("unknown HTTPEndpoint.URL.Scheme")
	errUnknownHTTPEndpointNetwork   = errors.New("unknown HTTPEndpoint.Network")
)

// HTTPPreparedRequest is a suspended request that only awaits
// for you to Resume it to deliver a result.
type HTTPPreparedRequest struct {
	resp *http.Response
	m    *HTTPEndpointMeasurement
	err  error
}

// Resume resumes the request and yields either a response or an error. You
// shall not call this function more than once.
func (r *HTTPPreparedRequest) Resume() (*http.Response, error) {
	return r.resp, r.err
}

// Measurement returns the associated measurement.
func (r *HTTPPreparedRequest) Measurement() *HTTPEndpointMeasurement {
	return r.m
}

// HTTPEndpointPrepareGet prepares a GET request for an HTTP endpoint.
//
// This prepared request WILL NOT follow redirects. If there is a redirect
// you will see it inside the specific database table.
//
// Arguments:
//
// - ctx is the context allowing to timeout the operation;
//
// - epnt is the HTTP endpoint;
//
// - jar is the cookie jar to use.
//
// Returns either a prepared request or an error.
func (mx *Measurer) HTTPEndpointPrepareGet(ctx context.Context,
	epnt *HTTPEndpoint, jar http.CookieJar) *HTTPPreparedRequest {
	out := &HTTPPreparedRequest{}
	out.resp, out.m, out.err = mx.httpEndpointGet(ctx, epnt, jar)
	return out
}

// httpEndpointGet implements HTTPEndpointGet.
func (mx *Measurer) httpEndpointGet(ctx context.Context, epnt *HTTPEndpoint,
	jar http.CookieJar) (*http.Response, *HTTPEndpointMeasurement, error) {
	resp, m, err := mx.httpEndpointGetMeasurement(ctx, epnt, jar)
	out := &HTTPEndpointMeasurement{
		URL: epnt.URL.String(),
		Endpoint: (&Endpoint{
			Network: epnt.Network,
			Address: epnt.Address,
		}).String(),
		Measurement: m,
	}
	return resp, out, err
}

// httpEndpointGetMeasurement implements httpEndpointGet.
//
// This function returns a triple where:
//
// - the first element is a valid response on success a nil response on failure
//
// - the second element is always a valid Measurement
//
// - the third element is a nil error on success and an error on failure
func (mx *Measurer) httpEndpointGetMeasurement(ctx context.Context, epnt *HTTPEndpoint,
	jar http.CookieJar) (resp *http.Response, m *Measurement, err error) {
	db := &MeasurementDB{}
	switch epnt.Network {
	case NetworkQUIC:
		resp, err = mx.httpEndpointGetQUIC(ctx, db, epnt, jar)
		m = db.AsMeasurement()
	case NetworkTCP:
		resp, err = mx.httpEndpointGetTCP(ctx, db, epnt, jar)
		m = db.AsMeasurement()
	default:
		m, err = &Measurement{}, errUnknownHTTPEndpointNetwork
	}
	return
}

// httpEndpointGetTCP specializes HTTPSEndpointGet for HTTP and HTTPS.
func (mx *Measurer) httpEndpointGetTCP(ctx context.Context,
	db WritableDB, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	switch epnt.URL.Scheme {
	case "http":
		return mx.httpEndpointGetHTTP(ctx, db, epnt, jar)
	case "https":
		return mx.httpEndpointGetHTTPS(ctx, db, epnt, jar)
	default:
		return nil, errUnknownHTTPEndpointURLScheme
	}
}

// httpEndpointGetHTTP specializes httpEndpointGetTCP for HTTP.
func (mx *Measurer) httpEndpointGetHTTP(ctx context.Context,
	db WritableDB, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header
	conn, err := mx.tcpConnect(ctx, db, epnt.Address)
	if err != nil {
		return nil, err
	}
	defer conn.Close() // we own it
	clnt := NewHTTPClientWithoutRedirects(db, jar,
		mx.NewHTTPTransportWithConn(mx.Logger, db, conn))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt, req)
}

// httpEndpointGetHTTPS specializes httpEndpointGetTCP for HTTPS.
func (mx *Measurer) httpEndpointGetHTTPS(ctx context.Context,
	db WritableDB, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header
	conn, err := mx.tlsConnectAndHandshake(ctx, db, epnt.Address, &tls.Config{
		ServerName: epnt.SNI,
		NextProtos: epnt.ALPN,
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close() // we own it
	clnt := NewHTTPClientWithoutRedirects(db, jar,
		mx.NewHTTPTransportWithTLSConn(mx.Logger, db, conn))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt, req)
}

// httpEndpointGetQUIC specializes httpEndpointGetTCP for QUIC.
func (mx *Measurer) httpEndpointGetQUIC(ctx context.Context,
	db WritableDB, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header
	sess, err := mx.quicHandshake(ctx, db, epnt.Address, &tls.Config{
		ServerName: epnt.SNI,
		NextProtos: epnt.ALPN,
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	if err != nil {
		return nil, err
	}
	// TODO(bassosimone): close session with correct message
	defer sess.CloseWithError(0, "") // we own it
	clnt := NewHTTPClientWithoutRedirects(db, jar,
		mx.NewHTTPTransportWithQUICSess(mx.Logger, db, sess))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt, req)
}

func (mx *Measurer) httpClientDo(ctx context.Context, clnt HTTPClient,
	epnt *HTTPEndpoint, req *http.Request) (*http.Response, error) {
	const timeout = 15 * time.Second
	ol := newOperationLogger(mx.Logger,
		"%s %s with %s/%s", req.Method, req.URL.String(), epnt.Address, epnt.Network)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := clnt.Do(req.WithContext(ctx))
	ol.Stop(err)
	return resp, err
}

// HTTPEndpointGetParallel performs an HTTPEndpointGet for each
// input endpoint using a pool of background goroutines.
//
// This function returns to the caller a channel where to read
// measurements from. The channel is closed when done.
func (mx *Measurer) HTTPEndpointGetParallel(ctx context.Context,
	jar http.CookieJar, epnts ...*HTTPEndpoint) <-chan *HTTPEndpointMeasurement {
	var (
		done   = make(chan interface{})
		input  = make(chan *HTTPEndpoint)
		output = make(chan *HTTPEndpointMeasurement)
	)
	go func() {
		defer close(input)
		for _, epnt := range epnts {
			input <- epnt
		}
	}()
	const parallelism = 3
	for i := 0; i < parallelism; i++ {
		go func() {
			for epnt := range input {
				output <- mx.HTTPEndpointGet(ctx, epnt, jar)
			}
			done <- true
		}()
	}
	go func() {
		for i := 0; i < parallelism; i++ {
			<-done
		}
		close(output)
	}()
	return output
}

// ResolverNetwork identifies the network of a resolver.
type ResolverNetwork string

var (
	// ResolverSystem is the system resolver (i.e., getaddrinfo)
	ResolverSystem = ResolverNetwork("system")

	// ResolverUDP is a resolver using DNS-over-UDP
	ResolverUDP = ResolverNetwork("udp")

	// ResolverForeign is a resolver that is not managed by
	// this package. We can wrap it, but we don't be able to
	// observe any event but Lookup{Host,HTTPSvc}
	ResolverForeign = ResolverNetwork("foreign")
)

// ResolverInfo contains info about a DNS resolver.
type ResolverInfo struct {
	// Network is the resolver's network (e.g., "doh", "udp")
	Network ResolverNetwork

	// Address is the address (e.g., "1.1.1.1:53", "https://1.1.1.1/dns-query")
	Address string

	// ForeignResolver is only used when Network's
	// value equals the ResolverForeign constant.
	ForeignResolver Resolver
}

// LookupURLHostParallel performs an LookupHost-like operation for each
// resolver that you provide as argument using a pool of goroutines.
func (mx *Measurer) LookupURLHostParallel(ctx context.Context,
	URL *url.URL, resos ...*ResolverInfo) <-chan *DNSMeasurement {
	var (
		done      = make(chan interface{})
		resolvers = make(chan *ResolverInfo)
		output    = make(chan *DNSMeasurement)
	)
	go func() {
		defer close(resolvers)
		for _, reso := range resos {
			resolvers <- reso
		}
	}()
	const parallelism = 3
	for i := 0; i < parallelism; i++ {
		go func() {
			for reso := range resolvers {
				mx.lookupHostWithResolverInfo(ctx, reso, URL, output)
			}
			done <- true
		}()
	}
	go func() {
		for i := 0; i < parallelism; i++ {
			<-done
		}
		close(output)
	}()
	return output
}

// lookupHostWithResolverInfo performs a LookupHost-like
// operation using the given ResolverInfo.
func (mx *Measurer) lookupHostWithResolverInfo(
	ctx context.Context, reso *ResolverInfo, URL *url.URL,
	output chan<- *DNSMeasurement) {
	switch reso.Network {
	case ResolverSystem:
		output <- mx.LookupHostSystem(ctx, URL.Hostname())
	case ResolverUDP:
		output <- mx.LookupHostUDP(ctx, URL.Hostname(), reso.Address)
	case ResolverForeign:
		output <- mx.lookupHostForeign(ctx, URL.Hostname(), reso.ForeignResolver)
	default:
		return
	}
	switch URL.Scheme {
	case "https":
	default:
		return
	}
	switch reso.Network {
	case ResolverUDP:
		output <- mx.LookupHTTPSSvcUDP(ctx, URL.Hostname(), reso.Address)
	case ResolverForeign:
		output <- mx.lookupHTTPSSvcUDPForeign(ctx, URL.Hostname(), reso.ForeignResolver)
	}
}

// LookupHostParallel is like LookupURLHostParallel but we only
// have in input an hostname rather than a URL. As such, we cannot
// determine whether to perform HTTPSSvc lookups and so we aren't
// going to perform this kind of lookups in this case.
func (mx *Measurer) LookupHostParallel(
	ctx context.Context, hostname, port string) <-chan *DNSMeasurement {
	out := make(chan *DNSMeasurement)
	go func() {
		defer close(out)
		URL := &url.URL{
			Scheme: "", // so we don't see https and we don't try HTTPSSvc
			Host:   net.JoinHostPort(hostname, port),
		}
		for m := range mx.LookupURLHostParallel(ctx, URL) {
			out <- &DNSMeasurement{Domain: hostname, Measurement: m.Measurement}
		}
	}()
	return out
}

// MeasureURLHelper is a Test Helper that discovers additional
// endpoints after MeasureURL has finished discovering endpoints
// via the usual DNS mechanism. The MeasureURLHelper:
//
// - is used by experiments to call a real test helper, i.e.,
// a remote service providing extra endpoints
//
// - is used by test helpers to augment the set of endpoints
// discovered so far with the ones provided by a client.
type MeasureURLHelper interface {
	// LookupExtraHTTPEndpoints searches for extra HTTP endpoints
	// suitable for the given URL we're measuring.
	//
	// Arguments:
	//
	// - ctx is the context for timeout/cancellation/deadline
	//
	// - URL is the URL we're currently measuring
	//
	// - headers contains the HTTP headers we wish to use
	//
	// - epnts is the current list of endpoints
	//
	// This function SHOULD return a NEW list of extra endpoints
	// it discovered and SHOULD NOT merge the epnts endpoints with
	// extra endpoints it discovered. Therefore:
	//
	// - on any kind of error it MUST return nil, err
	//
	// - on success it MUST return the NEW endpoints it discovered
	//
	// It is the caller's responsibility to merge the NEW list of
	// endpoints with the ones it passed as argument.
	//
	// It is also the caller's responsibility to ENSURE that the
	// newly returned endpoints only use the few headers that our
	// test helper protocol allows one to set.
	LookupExtraHTTPEndpoints(ctx context.Context, URL *url.URL,
		headers http.Header, epnts ...*HTTPEndpoint) ([]*HTTPEndpoint, error)
}

// MeasureURL measures an HTTP or HTTPS URL. The DNS resolvers
// and the Test Helpers we use in this measurement are the ones
// configured into the database. The default is to use the system
// resolver and to use not Test Helper. Use RegisterWCTH and
// RegisterUDPResolvers (and other similar functions that have
// not been written at the moment of writing this note) to
// augment the set of resolvers and Test Helpers we use here.
//
// Arguments:
//
// - ctx is the context for timeout/cancellation
//
// - URL is the URL to measure
//
// - header contains the HTTP headers for the request
//
// - cookies contains the cookies we should use for measuring
// this URL and possibly future redirections.
//
// To create an empty set of cookies, use NewCookieJar. It's
// normal to have empty cookies at the beginning. If we follow
// extra redirections after this run then the cookie jar will
// contain the cookies for following the next redirection.
//
// We need cookies because a small amount of URLs does not
// redirect properly without cookies. This has been
// documented at https://github.com/ooni/probe/issues/1727.
func (mx *Measurer) MeasureURL(
	ctx context.Context, URL string, headers http.Header,
	cookies http.CookieJar) (*URLMeasurement, error) {
	mx.Logger.Infof("MeasureURL url=%s", URL)
	m := &URLMeasurement{URL: URL}
	begin := time.Now()
	defer func() { m.TotalRuntime = time.Since(begin) }()
	parsed, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	if len(mx.Resolvers) < 1 {
		return nil, errors.New("measurer: no configured resolver")
	}
	dnsBegin := time.Now()
	for dns := range mx.LookupURLHostParallel(ctx, parsed, mx.Resolvers...) {
		m.DNS = append(m.DNS, dns)
	}
	m.DNSRuntime = time.Since(dnsBegin)
	epnts, err := AllHTTPEndpointsForURL(parsed, headers, m.DNS...)
	if err != nil {
		return nil, err
	}
	if mx.MeasureURLHelper != nil {
		thBegin := time.Now()
		extraEpnts, _ := mx.MeasureURLHelper.LookupExtraHTTPEndpoints(
			ctx, parsed, headers, epnts...)
		epnts = removeDuplicateHTTPEndpoints(append(epnts, extraEpnts...)...)
		m.THRuntime = time.Since(thBegin)
		mx.enforceAllowedHeadersOnly(epnts)
	}
	epntRuntime := time.Now()
	for epnt := range mx.HTTPEndpointGetParallel(ctx, cookies, epnts...) {
		m.Endpoints = append(m.Endpoints, epnt)
	}
	mx.maybeQUICFollowUp(ctx, m, cookies, epnts...)
	m.EpntsRuntime = time.Since(epntRuntime)
	m.fillRedirects()
	return m, nil
}

// maybeQUICFollowUp checks whether we need to use Alt-Svc to check
// for QUIC. We query for HTTPSSvc but currently only Cloudflare
// implements this proposed standard. So, this function is
// where we take care of all the other servers implementing QUIC.
func (mx *Measurer) maybeQUICFollowUp(ctx context.Context,
	m *URLMeasurement, cookies http.CookieJar, epnts ...*HTTPEndpoint) {
	altsvc := []string{}
	for _, epnt := range m.Endpoints {
		// Check whether we have a QUIC handshake. If so, then
		// HTTPSSvc worked and we can stop here.
		if epnt.QUICHandshake != nil {
			return
		}
		for idx, rtrip := range epnt.HTTPRoundTrip {
			if rtrip.Response == nil {
				stdlog.Printf("malformed HTTPRoundTrip@%d: %+v", idx, rtrip)
				continue
			}
			if v := rtrip.Response.Headers.Get("alt-svc"); v != "" {
				altsvc = append(altsvc, v)
			}
		}
	}
	// syntax:
	//
	// Alt-Svc: clear
	// Alt-Svc: <protocol-id>=<alt-authority>; ma=<max-age>
	// Alt-Svc: <protocol-id>=<alt-authority>; ma=<max-age>; persist=1
	//
	// multiple entries may be separated by comma.
	//
	// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Alt-Svc
	for _, header := range altsvc {
		entries := strings.Split(header, ",")
		if len(entries) < 1 {
			continue
		}
		for _, entry := range entries {
			parts := strings.Split(entry, ";")
			if len(parts) < 1 {
				continue
			}
			if parts[0] == "h3=\":443\"" {
				mx.doQUICFollowUp(ctx, m, cookies, epnts...)
				return
			}
		}
	}
}

// doQUICFollowUp runs when we know there's QUIC support via Alt-Svc.
func (mx *Measurer) doQUICFollowUp(ctx context.Context,
	m *URLMeasurement, cookies http.CookieJar, epnts ...*HTTPEndpoint) {
	quicEpnts := []*HTTPEndpoint{}
	// do not mutate the existing list rather create a new one
	for _, epnt := range epnts {
		quicEpnts = append(quicEpnts, &HTTPEndpoint{
			Domain:  epnt.Domain,
			Network: NetworkQUIC,
			Address: epnt.Address,
			SNI:     epnt.SNI,
			ALPN:    []string{"h3"},
			URL:     epnt.URL,
			Header:  epnt.Header,
		})
	}
	for mquic := range mx.HTTPEndpointGetParallel(ctx, cookies, quicEpnts...) {
		m.Endpoints = append(m.Endpoints, mquic)
	}
}

func (mx *Measurer) enforceAllowedHeadersOnly(epnts []*HTTPEndpoint) {
	for _, epnt := range epnts {
		epnt.Header = mx.keepOnlyAllowedHeaders(epnt.Header)
	}
}

func (mx *Measurer) keepOnlyAllowedHeaders(header http.Header) (out http.Header) {
	out = http.Header{}
	for k, vv := range header {
		switch strings.ToLower(k) {
		case "accept", "accept-language", "cookie", "user-agent":
			for _, v := range vv {
				out.Add(k, v)
			}
		default:
			// ignore all the other headers
		}
	}
	return
}

// redirectionQueue is the type we use to manage the redirection
// queue and to follow a reasonable number of redirects.
type redirectionQueue struct {
	q   []string
	cnt int
}

func (r *redirectionQueue) append(URL ...string) {
	r.q = append(r.q, URL...)
}

func (r *redirectionQueue) popleft() (URL string) {
	URL = r.q[0]
	r.q = r.q[1:]
	return
}

func (r *redirectionQueue) empty() bool {
	return len(r.q) <= 0
}

func (r *redirectionQueue) redirectionsCount() int {
	return r.cnt
}

// MeasureURLAndFollowRedirections is like MeasureURL except
// that it _also_ follows all the HTTP redirections.
func (mx *Measurer) MeasureHTTPURLAndFollowRedirections(ctx context.Context,
	URL string, headers http.Header, cookies http.CookieJar) <-chan *URLMeasurement {
	out := make(chan *URLMeasurement)
	go func() {
		defer close(out)
		meas, err := mx.MeasureURL(ctx, URL, headers, cookies)
		if err != nil {
			mx.Logger.Warnf("mx.MeasureURL failed: %s", err.Error())
			return
		}
		out <- meas
		rq := &redirectionQueue{q: meas.RedirectURLs}
		const maxRedirects = 7
		for !rq.empty() && rq.redirectionsCount() < maxRedirects {
			URL = rq.popleft()
			meas, err = mx.MeasureURL(ctx, URL, headers, cookies)
			if err != nil {
				mx.Logger.Warnf("mx.MeasureURL failed: %s", err.Error())
				return
			}
			out <- meas
			rq.append(meas.RedirectURLs...)
		}
	}()
	return out
}
