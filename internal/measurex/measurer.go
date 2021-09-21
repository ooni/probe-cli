package measurex

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Measurer performs measurements.
type Measurer struct {
	// DB is the MANDATORY database to use.
	DB *DB

	// HTTPClient is the MANDATORY HTTP client for the WCTH.
	HTTPClient HTTPClient

	// Logger is the MANDATORY logger to use.
	Logger Logger

	// Origin is the MANDATORY measurements origin to use.
	Origin Origin

	// TLSHandshaker is the MANDATORY TLS handshaker.
	TLSHandshaker netxlite.TLSHandshaker
}

// NewMeasurerWithDefaultSettings creates a new Measurer
// instance using the most default settings.
func NewMeasurerWithDefaultSettings() *Measurer {
	db := NewDB(time.Now())
	return &Measurer{
		DB:            db,
		HTTPClient:    &http.Client{},
		Logger:        log.Log,
		Origin:        OriginProbe,
		TLSHandshaker: netxlite.NewTLSHandshakerStdlib(log.Log),
	}
}

func (mx *Measurer) nextMeasurement() int64 {
	return mx.DB.NextMeasurementID()
}

// LookupHostSystem performs a LookupHost using the system resolver.
func (mx *Measurer) LookupHostSystem(ctx context.Context, domain string) *Measurement {
	const timeout = 4 * time.Second
	ol := newOperationLogger(mx.Logger, "LookupHost %s with getaddrinfo", domain)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	mid := mx.nextMeasurement()
	r := NewResolverSystem(mid, mx.Origin, mx.DB, mx.Logger)
	defer r.CloseIdleConnections()
	_, err := r.LookupHost(ctx, domain)
	ol.Stop(err)
	return NewMeasurement(mx.DB, mid)
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
// Returns a Measurement.
func (mx *Measurer) LookupHostUDP(
	ctx context.Context, domain, address string) *Measurement {
	const timeout = 4 * time.Second
	ol := newOperationLogger(mx.Logger, "LookupHost %s with %s/udp", domain, address)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	mid := mx.nextMeasurement()
	r := NewResolverUDP(mid, mx.Origin, mx.DB, mx.Logger, address)
	defer r.CloseIdleConnections()
	_, err := r.LookupHost(ctx, domain)
	ol.Stop(err)
	return NewMeasurement(mx.DB, mid)
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
// Returns a Measurement.
func (mx *Measurer) LookupHTTPSSvcUDP(
	ctx context.Context, domain, address string) *Measurement {
	const timeout = 4 * time.Second
	ol := newOperationLogger(mx.Logger, "LookupHTTPSvc %s with %s/udp", domain, address)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	mid := mx.nextMeasurement()
	r := NewResolverUDP(mid, mx.Origin, mx.DB, mx.Logger, address)
	defer r.CloseIdleConnections()
	_, err := r.LookupHTTPSSvcWithoutRetry(ctx, domain)
	ol.Stop(err)
	return NewMeasurement(mx.DB, mid)
}

// TCPConnect establishes a connection with a TCP endpoint.
//
// Arguments:
//
// - ctx is the context allowing to timeout the connect;
//
// - address is the TCP endpoint address (e.g., "8.8.4.4:443").
//
// Returns a Measurement.
func (mx *Measurer) TCPConnect(ctx context.Context, address string) *Measurement {
	mid := mx.nextMeasurement()
	conn, _ := mx.tcpConnect(ctx, mid, address)
	measurement := NewMeasurement(mx.DB, mid)
	if conn != nil {
		conn.Close()
	}
	return measurement
}

// tcpConnect is like TCPConnect but does not create a new measurement.
func (mx *Measurer) tcpConnect(ctx context.Context,
	measurementID int64, address string) (Conn, error) {
	const timeout = 10 * time.Second
	ol := newOperationLogger(mx.Logger, "TCPConnect %s", address)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	d := NewDialerWithoutResolver(measurementID, mx.Origin, mx.DB, mx.Logger)
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
// Returns a Measurement.
func (mx *Measurer) TLSConnectAndHandshake(ctx context.Context,
	address string, config *tls.Config) *Measurement {
	mid := mx.nextMeasurement()
	conn, _ := mx.tlsConnectAndHandshake(ctx, mid, address, config)
	measurement := NewMeasurement(mx.DB, mid)
	if conn != nil {
		conn.Close()
	}
	return measurement
}

// tlsConnectAndHandshake is like TLSConnectAndHandshake
// but does not create a new measurement.
func (mx *Measurer) tlsConnectAndHandshake(ctx context.Context,
	measurementID int64, address string, config *tls.Config) (TLSConn, error) {
	conn, err := mx.tcpConnect(ctx, measurementID, address)
	if err != nil {
		return nil, err
	}
	const timeout = 10 * time.Second
	ol := newOperationLogger(mx.Logger,
		"TLSHandshake %s with sni=%s", address, config.ServerName)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	th := WrapTLSHandshaker(measurementID, mx.Origin, mx.DB, mx.TLSHandshaker)
	tlsConn, err := th.Handshake(ctx, conn, config)
	ol.Stop(err)
	return tlsConn, err
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
// Returns a Measurement.
func (mx *Measurer) QUICHandshake(ctx context.Context, address string,
	config *tls.Config) *Measurement {
	mid := mx.nextMeasurement()
	sess, _ := mx.quicHandshake(ctx, mid, address, config)
	measurement := NewMeasurement(mx.DB, mid)
	if sess != nil {
		// TODO(bassosimone): close session with correct message
		sess.CloseWithError(0, "")
	}
	return measurement
}

// quicHandshake is like QUICHandshake but does not create a new measurement.
func (mx *Measurer) quicHandshake(ctx context.Context, measurementID int64,
	address string, config *tls.Config) (QUICEarlySession, error) {
	const timeout = 10 * time.Second
	ol := newOperationLogger(mx.Logger,
		"QUICHandshake %s with sni=%s", address, config.ServerName)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	qd := WrapQUICDialer(measurementID, mx.Origin, mx.DB,
		netxlite.NewQUICDialerWithoutResolver(WrapQUICListener(
			measurementID, mx.Origin, mx.DB, netxlite.NewQUICListener()),
			mx.Logger,
		))
	defer qd.CloseIdleConnections()
	sess, err := qd.DialContext(ctx, address, config)
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
	ctx context.Context, epnt *HTTPEndpoint, jar http.CookieJar) *Measurement {
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
	m    *Measurement
	err  error
}

// Resume resumes the request and yields either a response or an error. You
// shall not call this function more than once.
func (r *HTTPPreparedRequest) Resume() (*http.Response, error) {
	return r.resp, r.err
}

// Measurement returns the associated measurement.
func (r *HTTPPreparedRequest) Measurement() *Measurement {
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
	jar http.CookieJar) (resp *http.Response, m *Measurement, err error) {
	mid := mx.nextMeasurement()
	switch epnt.Network {
	case NetworkQUIC:
		resp, err = mx.httpEndpointGetQUIC(ctx, mid, epnt, jar)
		m = NewMeasurement(mx.DB, mid)
	case NetworkTCP:
		resp, err = mx.httpEndpointGetTCP(ctx, mid, epnt, jar)
		m = NewMeasurement(mx.DB, mid)
	default:
		m, err = &Measurement{}, errUnknownHTTPEndpointNetwork
	}
	return
}

// httpEndpointGetTCP specializes HTTPSEndpointGet for HTTP and HTTPS.
func (mx *Measurer) httpEndpointGetTCP(ctx context.Context,
	measurementID int64, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	switch epnt.URL.Scheme {
	case "http":
		return mx.httpEndpointGetHTTP(ctx, measurementID, epnt, jar)
	case "https":
		return mx.httpEndpointGetHTTPS(ctx, measurementID, epnt, jar)
	default:
		return nil, errUnknownHTTPEndpointURLScheme
	}
}

// httpEndpointGetHTTP specializes httpEndpointGetTCP for HTTP.
func (mx *Measurer) httpEndpointGetHTTP(ctx context.Context,
	measurementID int64, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header
	conn, err := mx.tcpConnect(ctx, measurementID, epnt.Address)
	if err != nil {
		return nil, err
	}
	defer conn.Close() // we own it
	clnt := NewHTTPClientWithoutRedirects(measurementID, mx.Origin, mx.DB, jar,
		NewHTTPTransportWithConn(measurementID, mx.Origin, mx.Logger, mx.DB, conn))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt, req)
}

// httpEndpointGetHTTPS specializes httpEndpointGetTCP for HTTPS.
func (mx *Measurer) httpEndpointGetHTTPS(ctx context.Context,
	measurementID int64, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header
	conn, err := mx.tlsConnectAndHandshake(ctx, measurementID, epnt.Address, &tls.Config{
		ServerName: epnt.SNI,
		NextProtos: epnt.ALPN,
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close() // we own it
	clnt := NewHTTPClientWithoutRedirects(measurementID, mx.Origin, mx.DB, jar,
		NewHTTPTransportWithTLSConn(measurementID, mx.Origin, mx.Logger, mx.DB, conn))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt, req)
}

// httpEndpointGetQUIC specializes httpEndpointGetTCP for QUIC.
func (mx *Measurer) httpEndpointGetQUIC(ctx context.Context,
	measurementID int64, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header
	sess, err := mx.quicHandshake(ctx, measurementID, epnt.Address, &tls.Config{
		ServerName: epnt.SNI,
		NextProtos: epnt.ALPN,
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	if err != nil {
		return nil, err
	}
	// TODO(bassosimone): close session with correct message
	defer sess.CloseWithError(0, "") // we own it
	clnt := NewHTTPClientWithoutRedirects(measurementID, mx.Origin, mx.DB, jar,
		NewHTTPTransportWithQUICSess(measurementID, mx.Origin, mx.Logger, mx.DB, sess))
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

// LookupWCTH performs an Endpoint lookup using the WCTH (i.e.,
// the Web Connectivity Test Helper) web service.
//
// Arguments:
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
// This function will safely discard any non-TCP endpoints
// in the input list and will only use TCP endpoints.
//
// Returns a measurement.
func (mx *Measurer) LookupWCTH(ctx context.Context, URL *url.URL,
	endpoints []*Endpoint, port string, WCTHURL string) *Measurement {
	const timeout = 30 * time.Second
	ol := newOperationLogger(mx.Logger, "WCTH %s with %s", URL.String(), WCTHURL)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	mid := mx.nextMeasurement()
	w := NewWCTHWorker(mid, mx.Logger, mx.DB, mx.HTTPClient, WCTHURL)
	_, err := w.Run(ctx, URL, mx.onlyTCPEndpoints(endpoints))
	ol.Stop(err)
	return NewMeasurement(mx.DB, mid)
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

// HTTPEndpointGetParallel performs an HTTPEndpointGet for each
// input endpoint using a pool of background goroutines.
//
// This function returns to the caller a channel where to read
// measurements from. The channel is closed when done.
func (mx *Measurer) HTTPEndpointGetParallel(ctx context.Context,
	jar http.CookieJar, epnts ...*HTTPEndpoint) <-chan *Measurement {
	var (
		done   = make(chan interface{})
		input  = make(chan *HTTPEndpoint)
		output = make(chan *Measurement)
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

// RegisterUDPResolvers registers UDP resolvers into the DB.
func (mx *Measurer) RegisterUDPResolvers(resolvers ...string) {
	for _, resolver := range resolvers {
		mx.DB.InsertIntoResolvers("udp", resolver)
	}
}

// LookupURLHostParallel performs an LookupHost-like operation for each
// DNS resolver registered into the database using a pool of background
// goroutines.
//
// This function returns to the caller a channel where to read
// measurements from. The channel is closed when done.
func (mx *Measurer) LookupURLHostParallel(
	ctx context.Context, URL *url.URL) <-chan *Measurement {
	var (
		done      = make(chan interface{})
		resolvers = make(chan *ResolverInfo)
		output    = make(chan *Measurement)
	)
	go func() {
		defer close(resolvers)
		for _, reso := range mx.DB.SelectAllFromResolvers() {
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
	output chan<- *Measurement) {
	switch reso.Network {
	case "system":
		output <- mx.LookupHostSystem(ctx, URL.Hostname())
	case "udp":
		output <- mx.LookupHostUDP(ctx, URL.Hostname(), reso.Address)
	default:
		return
	}
	switch URL.Scheme {
	case "https":
	default:
		return
	}
	switch reso.Network {
	case "udp":
		output <- mx.LookupHTTPSSvcUDP(ctx, URL.Hostname(), reso.Address)
	}
}

// LookupHostParallel is like LookupURLHostParallel but we only
// have in input an hostname rather than a URL. As such, we cannot
// determine whether to perform HTTPSSvc lookups and so we aren't
// going to perform this kind of lookups in this case.
func (mx *Measurer) LookupHostParallel(
	ctx context.Context, hostname, port string) <-chan *Measurement {
	return mx.LookupURLHostParallel(ctx, &url.URL{
		Scheme: "", // so we don't see https and we don't try HTTPSSvc
		Host:   net.JoinHostPort(hostname, port),
	})
}

// RegisterWCTH registers URLs for the WCTH.
func (mx *Measurer) RegisterWCTH(URLs ...string) {
	for _, URL := range URLs {
		mx.DB.InsertIntoTestHelpers("wcth", URL)
	}
}

// QueryTestHelperParallel performs a parallel query for the
// given URL to all known test helpers.
func (mx *Measurer) QueryTestHelperParallel(
	ctx context.Context, URL *url.URL) <-chan *Measurement {
	var (
		done   = make(chan interface{})
		ths    = make(chan *TestHelperInfo)
		output = make(chan *Measurement)
	)
	go func() {
		defer close(ths)
		for _, th := range mx.DB.SelectAllFromTestHelpers() {
			ths <- th
		}
	}()
	const parallelism = 1 // maybe raise in the future?
	for i := 0; i < parallelism; i++ {
		go func() {
			for th := range ths {
				mx.asyncTestHelperQuery(ctx, th, URL, output)
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

func (mx *Measurer) asyncTestHelperQuery(
	ctx context.Context, th *TestHelperInfo, URL *url.URL,
	output chan<- *Measurement) {
	switch th.Protocol {
	case "wcth":
		port, err := PortFromURL(URL)
		if err != nil {
			return // TODO(bassosimone): what to do about this error?
		}
		endpoints := mx.DB.SelectAllEndpointsForDomain(URL.Hostname(), port)
		output <- mx.LookupWCTH(ctx, URL, endpoints, port, th.URL)
	default:
		// don't know what to do
	}
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
	ctx context.Context, URL string, cookies http.CookieJar) *URLMeasurement {
	mx.Logger.Infof("MeasureURL url=%s", URL)
	m := &URLMeasurement{URL: URL}
	begin := time.Now()
	defer func() { m.TotalRuntime = time.Since(begin) }()
	parsed, err := url.Parse(URL)
	if err != nil {
		m.CannotParseURL = true
		return m
	}
	dnsBegin := time.Now()
	for dns := range mx.LookupURLHostParallel(ctx, parsed) {
		m.DNS = append(m.DNS, dns)
	}
	m.DNSRuntime = time.Since(dnsBegin)
	thBegin := time.Now()
	for th := range mx.QueryTestHelperParallel(ctx, parsed) {
		m.TH = append(m.TH, th)
	}
	m.THRuntime = time.Since(thBegin)
	epnts, err := mx.DB.SelectAllHTTPEndpointsForURL(parsed)
	if err != nil {
		m.CannotGenerateEndpoints = true
		return m
	}
	epntRuntime := time.Now()
	for epnt := range mx.HTTPEndpointGetParallel(ctx, cookies, epnts...) {
		m.Endpoints = append(m.Endpoints, epnt)
	}
	m.EpntsRuntime = time.Since(epntRuntime)
	m.fillRedirects()
	return m
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
	URL string, cookies http.CookieJar) <-chan *URLMeasurement {
	out := make(chan *URLMeasurement)
	go func() {
		defer close(out)
		m := mx.MeasureURL(ctx, URL, cookies)
		out <- m
		rq := &redirectionQueue{q: m.RedirectURLs}
		const maxRedirects = 7
		for !rq.empty() && rq.redirectionsCount() < maxRedirects {
			URL = rq.popleft()
			m = mx.MeasureURL(ctx, URL, cookies)
			out <- m
			rq.append(m.RedirectURLs...)
		}
	}()
	return out
}
