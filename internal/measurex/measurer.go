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
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Measurer performs measurements. If you don't use a factory
// for creating this type, make sure you set all the MANDATORY fields.
type Measurer struct {
	// Begin is when we started measuring (this field is MANDATORY).
	Begin time.Time

	// DNSLookupTimeout is the OPTIONAL timeout for performing
	// a DNS lookup. If not set, we use a default value.
	//
	// Note that the underlying network implementation MAY use a
	// shorter-than-you-selected watchdog timeout. In such a case,
	// the shorter watchdog timeout will prevail.
	DNSLookupTimeout time.Duration

	// HTTPClient is the MANDATORY HTTP client for the WCTH.
	HTTPClient model.HTTPClient

	// HTTPMaxBodySnapshotSize is the OPTIONAL maximum size,
	// in bytes, of the response body snapshot we save. If this field
	// is zero or negative, we'll use a small default value.
	HTTPMaxBodySnapshotSize int64

	// HTTPRoundTripTimeout is the OPTIONAL timeout for performing
	// an HTTP round trip. If not set, we use a default value.
	//
	// Note that the underlying network implementation MAY use a
	// shorter-than-you-selected watchdog timeout. In such a case,
	// the shorter watchdog timeout will prevail.
	HTTPRoundTripTimeout time.Duration

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// MeasureURLHelper is the OPTIONAL test helper to use when
	// we're measuring using the MeasureURL function. If this field
	// is not set, we'll not be using any helper.
	MeasureURLHelper MeasureURLHelper

	// QUICHandshakeTimeout is the OPTIONAL timeout for performing
	// a QUIC handshake. If not set, we use a default value.
	//
	// Note that the underlying network implementation MAY use a
	// shorter-than-you-selected watchdog timeout. In such a case,
	// the shorter watchdog timeout will prevail.
	QUICHandshakeTimeout time.Duration

	// Resolvers is the MANDATORY list of resolvers.
	Resolvers []*ResolverInfo

	// TCPConnectTimeout is the OPTIONAL timeout for performing
	// a tcp connect. If not set, we use a default value.
	//
	// Note that the underlying network implementation MAY use a
	// shorter-than-you-selected watchdog timeout. In such a case,
	// the shorter watchdog timeout will prevail.
	TCPconnectTimeout time.Duration

	// TLSHandshakeTimeout is the OPTIONAL timeout for performing
	// a tls handshake. If not set, we use a default value.
	//
	// Note that the underlying network implementation MAY use a
	// shorter-than-you-selected watchdog timeout. In such a case,
	// the shorter watchdog timeout will prevail.
	TLSHandshakeTimeout time.Duration

	// TLSHandshaker is the MANDATORY TLS handshaker.
	TLSHandshaker model.TLSHandshaker
}

// NewMeasurerWithDefaultSettings creates a new Measurer
// instance using the most default settings.
func NewMeasurerWithDefaultSettings() *Measurer {
	return &Measurer{
		Begin:                   time.Now(),
		DNSLookupTimeout:        0,
		HTTPClient:              &http.Client{},
		HTTPMaxBodySnapshotSize: 0,
		HTTPRoundTripTimeout:    0,
		Logger:                  log.Log,
		MeasureURLHelper:        nil,
		QUICHandshakeTimeout:    0,
		Resolvers: []*ResolverInfo{{
			Network: "system",
			Address: "",
		}, {
			Network: "udp",
			Address: "8.8.4.4:53",
		}},
		TCPconnectTimeout:   0,
		TLSHandshakeTimeout: 0,
		TLSHandshaker:       netxlite.NewTLSHandshakerStdlib(log.Log),
	}
}

// DefaultDNSLookupTimeout is the default DNS lookup timeout.
const DefaultDNSLookupTimeout = 4 * time.Second

// dnsLookupTimeout selects the correct DNS lookup timeout.
func (mx *Measurer) dnsLookupTimeout() time.Duration {
	if mx.DNSLookupTimeout > 0 {
		return mx.DNSLookupTimeout
	}
	return DefaultDNSLookupTimeout
}

// LookupHostSystem performs a LookupHost using the system resolver.
func (mx *Measurer) LookupHostSystem(ctx context.Context, domain string) *DNSMeasurement {
	timeout := mx.dnsLookupTimeout()
	ol := NewOperationLogger(mx.Logger, "LookupHost %s with getaddrinfo", domain)
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
	ctx context.Context, domain string, r model.Resolver) *DNSMeasurement {
	timeout := mx.dnsLookupTimeout()
	ol := NewOperationLogger(mx.Logger, "LookupHost %s with %s", domain, r.Network())
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
	timeout := mx.dnsLookupTimeout()
	ol := NewOperationLogger(mx.Logger, "LookupHost %s with %s/udp", domain, address)
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
	timeout := mx.dnsLookupTimeout()
	ol := NewOperationLogger(mx.Logger, "LookupHTTPSvc %s with %s/udp", domain, address)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	r := mx.NewResolverUDP(db, mx.Logger, address)
	defer r.CloseIdleConnections()
	_, err := r.LookupHTTPS(ctx, domain)
	ol.Stop(err)
	return &DNSMeasurement{
		Domain:      domain,
		Measurement: db.AsMeasurement(),
	}
}

// lookupHTTPSSvcUDPForeign is like LookupHTTPSSvcUDP
// except that it uses a "foreign" resolver.
func (mx *Measurer) lookupHTTPSSvcUDPForeign(
	ctx context.Context, domain string, r model.Resolver) *DNSMeasurement {
	timeout := mx.dnsLookupTimeout()
	ol := NewOperationLogger(mx.Logger, "LookupHTTPSvc %s with %s", domain, r.Address())
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	_, err := mx.WrapResolver(db, r).LookupHTTPS(ctx, domain)
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
	conn, _ := mx.TCPConnectWithDB(ctx, db, address)
	measurement := db.AsMeasurement()
	if conn != nil {
		conn.Close()
	}
	return &EndpointMeasurement{
		Network:     NetworkTCP,
		Address:     address,
		Measurement: measurement,
	}
}

// DefaultTCPConnectTimeout is the default TCP connect timeout.
const DefaultTCPConnectTimeout = 15 * time.Second

// tcpConnectTimeout selects the correct TCP connect timeout.
func (mx *Measurer) tcpConnectTimeout() time.Duration {
	if mx.TCPconnectTimeout > 0 {
		return mx.TCPconnectTimeout
	}
	return DefaultTCPConnectTimeout
}

// TCPConnectWithDB is like TCPConnect but does not create a new measurement,
// rather it just stores the events inside of the given DB.
func (mx *Measurer) TCPConnectWithDB(ctx context.Context, db WritableDB, address string) (Conn, error) {
	timeout := mx.tcpConnectTimeout()
	ol := NewOperationLogger(mx.Logger, "TCPConnect %s", address)
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
	conn, _ := mx.TLSConnectAndHandshakeWithDB(ctx, db, address, config)
	measurement := db.AsMeasurement()
	if conn != nil {
		conn.Close()
	}
	return &EndpointMeasurement{
		Network:     NetworkTCP,
		Address:     address,
		Measurement: measurement,
	}
}

// DefaultTLSHandshakeTimeout is the default TLS handshake timeout.
const DefaultTLSHandshakeTimeout = 10 * time.Second

// tlsHandshakeTimeout selects the correct TLS handshake timeout.
func (mx *Measurer) tlsHandshakeTimeout() time.Duration {
	if mx.TLSHandshakeTimeout > 0 {
		return mx.TLSHandshakeTimeout
	}
	return DefaultTLSHandshakeTimeout
}

// TLSConnectAndHandshakeWithDB is like TLSConnectAndHandshake but
// uses the given DB instead of creating a new Measurement.
func (mx *Measurer) TLSConnectAndHandshakeWithDB(ctx context.Context,
	db WritableDB, address string, config *tls.Config) (netxlite.TLSConn, error) {
	conn, err := mx.TCPConnectWithDB(ctx, db, address)
	if err != nil {
		return nil, err
	}
	timeout := mx.tlsHandshakeTimeout()
	ol := NewOperationLogger(mx.Logger,
		"TLSHandshake %s with sni=%s", address, config.ServerName)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	th := mx.WrapTLSHandshaker(db, mx.TLSHandshaker)
	tlsConn, _, err := th.Handshake(ctx, conn, config)
	ol.Stop(err)
	if err != nil {
		return nil, err
	}
	// cast safe according to the docs of netxlite's handshaker
	return tlsConn.(netxlite.TLSConn), nil
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
	qconn, _ := mx.QUICHandshakeWithDB(ctx, db, address, config)
	measurement := db.AsMeasurement()
	if qconn != nil {
		// TODO(bassosimone): close connection with correct message
		qconn.CloseWithError(0, "")
	}
	return &EndpointMeasurement{
		Network:     NetworkQUIC,
		Address:     address,
		Measurement: measurement,
	}
}

// DefaultQUICHandshakeTimeout is the default QUIC handshake timeout.
const DefaultQUICHandshakeTimeout = 10 * time.Second

// quicHandshakeTimeout selects the correct QUIC handshake timeout.
func (mx *Measurer) quicHandshakeTimeout() time.Duration {
	if mx.QUICHandshakeTimeout > 0 {
		return mx.QUICHandshakeTimeout
	}
	return DefaultQUICHandshakeTimeout
}

// QUICHandshakeWithDB is like QUICHandshake but uses the given
// db to store events rather than creating a temporary one and
// use it to generate a new Measurement.
func (mx *Measurer) QUICHandshakeWithDB(ctx context.Context, db WritableDB,
	address string, config *tls.Config) (quic.EarlyConnection, error) {
	timeout := mx.quicHandshakeTimeout()
	ol := NewOperationLogger(mx.Logger,
		"QUICHandshake %s with sni=%s", address, config.ServerName)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	qd := mx.NewQUICDialerWithoutResolver(db, mx.Logger)
	defer qd.CloseIdleConnections()
	qconn, err := qd.DialContext(ctx, "udp", address, config, &quic.Config{})
	ol.Stop(err)
	return qconn, err
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

// HTTPEndpointGetWithoutCookies is like HTTPEndpointGet
// but does not require you to provide a CookieJar.
func (mx *Measurer) HTTPEndpointGetWithoutCookies(
	ctx context.Context, epnt *HTTPEndpoint) *HTTPEndpointMeasurement {
	return mx.HTTPEndpointGet(ctx, epnt, NewCookieJar())
}

var (
	errUnknownHTTPEndpointURLScheme = errors.New("unknown HTTPEndpoint.URL.Scheme")

	// ErrUnknownHTTPEndpointNetwork means that the given endpoint's
	// network is of a type that we don't know how to handle.
	ErrUnknownHTTPEndpointNetwork = errors.New("unknown HTTPEndpoint.Network")
)

// httpEndpointGet implements HTTPEndpointGet.
func (mx *Measurer) httpEndpointGet(ctx context.Context, epnt *HTTPEndpoint,
	jar http.CookieJar) (*http.Response, *HTTPEndpointMeasurement, error) {
	resp, m, err := mx.httpEndpointGetMeasurement(ctx, epnt, jar)
	out := &HTTPEndpointMeasurement{
		URL:         epnt.URL.String(),
		Network:     epnt.Network,
		Address:     epnt.Address,
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
	resp, err = mx.httpEndpointGetWithDB(ctx, epnt, db, jar)
	m = db.AsMeasurement()
	return
}

// HTTPEndpointGetWithDB is an HTTPEndpointGet that stores the
// events into the given WritableDB.
func (mx *Measurer) HTTPEndpointGetWithDB(ctx context.Context, epnt *HTTPEndpoint,
	db WritableDB, jar http.CookieJar) (err error) {
	switch epnt.Network {
	case NetworkQUIC:
		_, err = mx.httpEndpointGetQUIC(ctx, db, epnt, jar)
	case NetworkTCP:
		_, err = mx.httpEndpointGetTCP(ctx, db, epnt, jar)
	default:
		err = ErrUnknownHTTPEndpointNetwork
	}
	return
}

// httpEndpointGetWithDB is an HTTPEndpointGet that stores the
// events into the given WritableDB.
func (mx *Measurer) httpEndpointGetWithDB(ctx context.Context, epnt *HTTPEndpoint,
	db WritableDB, jar http.CookieJar) (resp *http.Response, err error) {
	switch epnt.Network {
	case NetworkQUIC:
		resp, err = mx.httpEndpointGetQUIC(ctx, db, epnt, jar)
	case NetworkTCP:
		resp, err = mx.httpEndpointGetTCP(ctx, db, epnt, jar)
	default:
		err = ErrUnknownHTTPEndpointNetwork
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
	conn, err := mx.TCPConnectWithDB(ctx, db, epnt.Address)
	if err != nil {
		return nil, err
	}
	defer conn.Close() // we own it
	clnt := NewHTTPClientWithoutRedirects(db, jar,
		mx.NewHTTPTransportWithConn(mx.Logger, db, conn))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt)
}

// httpEndpointGetHTTPS specializes httpEndpointGetTCP for HTTPS.
func (mx *Measurer) httpEndpointGetHTTPS(ctx context.Context,
	db WritableDB, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	conn, err := mx.TLSConnectAndHandshakeWithDB(ctx, db, epnt.Address, &tls.Config{
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
	return mx.httpClientDo(ctx, clnt, epnt)
}

// httpEndpointGetQUIC specializes httpEndpointGetTCP for QUIC.
func (mx *Measurer) httpEndpointGetQUIC(ctx context.Context,
	db WritableDB, epnt *HTTPEndpoint, jar http.CookieJar) (*http.Response, error) {
	qconn, err := mx.QUICHandshakeWithDB(ctx, db, epnt.Address, &tls.Config{
		ServerName: epnt.SNI,
		NextProtos: epnt.ALPN,
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	if err != nil {
		return nil, err
	}
	// TODO(bassosimone): close connection with correct message
	defer qconn.CloseWithError(0, "") // we own it
	clnt := NewHTTPClientWithoutRedirects(db, jar,
		mx.NewHTTPTransportWithQUICConn(mx.Logger, db, qconn))
	defer clnt.CloseIdleConnections()
	return mx.httpClientDo(ctx, clnt, epnt)
}

// HTTPClientGET performs a GET operation of the given URL
// using the given HTTP client instance.
func (mx *Measurer) HTTPClientGET(
	ctx context.Context, clnt model.HTTPClient, URL *url.URL) (*http.Response, error) {
	return mx.httpClientDo(ctx, clnt, &HTTPEndpoint{
		Domain:  URL.Hostname(),
		Network: "tcp",
		Address: URL.Hostname(),
		SNI:     "",         // not needed
		ALPN:    []string{}, // not needed
		URL:     URL,
		Header:  NewHTTPRequestHeaderForMeasuring(),
	})
}

// DefaultHTTPRoundTripTimeout is the default HTTP round-trip timeout.
const DefaultHTTPRoundTripTimeout = 15 * time.Second

// httpRoundTripTimeout selects the correct HTTP round-trip timeout.
func (mx *Measurer) httpRoundTripTimeout() time.Duration {
	if mx.HTTPRoundTripTimeout > 0 {
		return mx.HTTPRoundTripTimeout
	}
	return DefaultHTTPRoundTripTimeout
}

func (mx *Measurer) httpClientDo(ctx context.Context,
	clnt model.HTTPClient, epnt *HTTPEndpoint) (*http.Response, error) {
	req, err := NewHTTPGetRequest(ctx, epnt.URL.String())
	if err != nil {
		return nil, err
	}
	req.Header = epnt.Header.Clone() // must clone because of parallel usage
	timeout := mx.httpRoundTripTimeout()
	ol := NewOperationLogger(mx.Logger,
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
// You can choose the parallelism with the parallelism argument. If this
// argument is zero, or negative, we use a small default value.
//
// This function returns to the caller a channel where to read
// measurements from. The channel is closed when done.
func (mx *Measurer) HTTPEndpointGetParallel(ctx context.Context, parallelism int,
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
	if parallelism <= 0 {
		parallelism = 3
	}
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
	ForeignResolver model.Resolver
}

// LookupURLHostParallel performs an LookupHost-like operation for each
// resolver that you provide as argument using a pool of goroutines.
//
// You can choose the parallelism with the parallelism argument. If this
// argument is zero, or negative, we use a small default value.
func (mx *Measurer) LookupURLHostParallel(ctx context.Context, parallelism int,
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
	if parallelism <= 0 {
		parallelism = 3
	}
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
//
// You can choose the parallelism with the parallelism argument. If this
// argument is zero, or negative, we use a small default value.
func (mx *Measurer) LookupHostParallel(ctx context.Context,
	parallelism int, hostname, port string) <-chan *DNSMeasurement {
	out := make(chan *DNSMeasurement)
	go func() {
		defer close(out)
		URL := &url.URL{
			Scheme: "", // so we don't see https and we don't try HTTPSSvc
			Host:   net.JoinHostPort(hostname, port),
		}
		for m := range mx.LookupURLHostParallel(ctx, parallelism, URL) {
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
	// as well as the TH measurement to be added to the measurement
	// that the URL measurer is constructing.
	//
	// It is the caller's responsibility to merge the NEW list of
	// endpoints with the ones it passed as argument.
	//
	// It is also the caller's responsibility to ENSURE that the
	// newly returned endpoints only use the few headers that our
	// test helper protocol allows one to set.
	LookupExtraHTTPEndpoints(ctx context.Context, URL *url.URL,
		headers http.Header, epnts ...*HTTPEndpoint) (
		newEpnts []*HTTPEndpoint, thMeasurement *THMeasurement, err error)
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
// - ctx is the context for timeout/cancellation.
//
// - parallelism is the number of parallel background goroutines
// to use to perform parallelizable operations (i.e., operations for
// which `measurex` defines an `OpParallel` API where `Op` is the
// name of an operation implemented by `measurex`). If parallel's value
// is zero or negative, we use a reasonably small default.
//
// - URL is the URL to measure.
//
// - header contains the HTTP headers for the request.
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
	ctx context.Context, parallelism int, URL string, headers http.Header,
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
	for dns := range mx.LookupURLHostParallel(ctx, parallelism, parsed, mx.Resolvers...) {
		m.DNS = append(m.DNS, dns)
	}
	m.DNSRuntime = time.Since(dnsBegin)
	epnts, err := AllHTTPEndpointsForURL(parsed, headers, m.DNS...)
	if err != nil {
		return nil, err
	}
	if mx.MeasureURLHelper != nil {
		thBegin := time.Now()
		extraEpnts, thMeasurement, _ := mx.MeasureURLHelper.LookupExtraHTTPEndpoints(
			ctx, parsed, headers, epnts...)
		m.THRuntime = time.Since(thBegin)
		epnts = removeDuplicateHTTPEndpoints(append(epnts, extraEpnts...)...)
		m.TH = thMeasurement
		mx.enforceAllowedHeadersOnly(epnts)
	}
	epntRuntime := time.Now()
	for epnt := range mx.HTTPEndpointGetParallel(ctx, parallelism, cookies, epnts...) {
		m.Endpoints = append(m.Endpoints, epnt)
	}
	switch parsed.Scheme {
	case "https":
		mx.maybeQUICFollowUp(ctx, parallelism, m, cookies, epnts...)
	default:
		// nothing to do
	}
	m.EpntsRuntime = time.Since(epntRuntime)
	m.fillRedirects()
	return m, nil
}

// maybeQUICFollowUp checks whether we need to use Alt-Svc to check
// for QUIC. We query for HTTPSSvc but currently only Cloudflare
// implements this proposed standard. So, this function is
// where we take care of all the other servers implementing QUIC.
func (mx *Measurer) maybeQUICFollowUp(ctx context.Context, parallelism int,
	m *URLMeasurement, cookies http.CookieJar, epnts ...*HTTPEndpoint) {
	altsvc := []string{}
	for _, epnt := range m.Endpoints {
		// Check whether we have a QUIC handshake. If so, then
		// HTTPSSvc worked and we can stop here.
		if epnt.QUICHandshake != nil {
			return
		}
		for _, rtrip := range epnt.HTTPRoundTrip {
			if v := rtrip.ResponseHeaders.Get("alt-svc"); v != "" {
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
				mx.doQUICFollowUp(ctx, parallelism, m, cookies, epnts...)
				return
			}
		}
	}
}

// doQUICFollowUp runs when we know there's QUIC support via Alt-Svc.
func (mx *Measurer) doQUICFollowUp(ctx context.Context, parallelism int,
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
	for mquic := range mx.HTTPEndpointGetParallel(ctx, parallelism, cookies, quicEpnts...) {
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
	r.cnt++
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
func (mx *Measurer) MeasureURLAndFollowRedirections(ctx context.Context, parallelism int,
	URL string, headers http.Header, cookies http.CookieJar) <-chan *URLMeasurement {
	out := make(chan *URLMeasurement)
	go func() {
		defer close(out)
		meas, err := mx.MeasureURL(ctx, parallelism, URL, headers, cookies)
		if err != nil {
			mx.Logger.Warnf("mx.MeasureURL failed: %s", err.Error())
			return
		}
		out <- meas
		rq := &redirectionQueue{q: meas.RedirectURLs}
		const maxRedirects = 7
		for !rq.empty() && rq.redirectionsCount() < maxRedirects {
			URL = rq.popleft()
			meas, err = mx.MeasureURL(ctx, parallelism, URL, headers, cookies)
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
