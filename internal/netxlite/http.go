package netxlite

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	oohttp "github.com/ooni/oohttp"
)

// HTTPTransport is an http.Transport-like structure.
type HTTPTransport interface {
	// RoundTrip performs the HTTP round trip.
	RoundTrip(req *http.Request) (*http.Response, error)

	// CloseIdleConnections closes idle connections.
	CloseIdleConnections()
}

// httpTransportLogger is an HTTPTransport with logging.
type httpTransportLogger struct {
	// HTTPTransport is the underlying HTTP transport.
	HTTPTransport HTTPTransport

	// Logger is the underlying logger.
	Logger Logger
}

var _ HTTPTransport = &httpTransportLogger{}

func (txp *httpTransportLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	req.Header.Set("Host", host) // anticipate what Go would do
	return txp.logTrip(req)
}

// logTrip is an HTTP round trip with logging.
func (txp *httpTransportLogger) logTrip(req *http.Request) (*http.Response, error) {
	txp.Logger.Debugf("> %s %s", req.Method, req.URL.String())
	for key, values := range req.Header {
		for _, value := range values {
			txp.Logger.Debugf("> %s: %s", key, value)
		}
	}
	txp.Logger.Debug(">")
	resp, err := txp.HTTPTransport.RoundTrip(req)
	if err != nil {
		txp.Logger.Debugf("< %s", err)
		return nil, err
	}
	txp.Logger.Debugf("< %d", resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			txp.Logger.Debugf("< %s: %s", key, value)
		}
	}
	txp.Logger.Debug("<")
	return resp, nil
}

func (txp *httpTransportLogger) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
}

// httpTransportConnectionsCloser is an HTTPTransport that
// correctly forwards CloseIdleConnections calls.
type httpTransportConnectionsCloser struct {
	HTTPTransport
	Dialer
	TLSDialer
}

// CloseIdleConnections forwards the CloseIdleConnections calls.
func (txp *httpTransportConnectionsCloser) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
	txp.Dialer.CloseIdleConnections()
	txp.TLSDialer.CloseIdleConnections()
}

// NewHTTPTransport creates a new HTTP transport using the given
// dialer and TLS dialer to create connections.
//
// The returned transport will use the given Logger for logging.
//
// The returned transport will gracefully handle TLS connections
// created using gitlab.com/yawning/utls.git.
//
// The returned transport will not have a configured proxy, not
// even the proxy configurable from the environment.
//
// The returned transport will disable transparent decompression
// of compressed response bodies (and will not automatically
// ask for such compression, though you can always do that manually).
//
// The returned transport will configure TCP and TLS connections
// created using its dialer and TLS dialer to always have a
// read watchdog timeout to address https://github.com/ooni/probe/issues/1609.
//
// The returned transport will always enforce 1 connection per host
// and we cannot get rid of this QUIRK requirement because it is
// necessary to perform sane measurements with tracing. We will be
// able to possibly relax this requirement after we change the
// way in which we perform measurements.
//
// The returned transport will set a default user agent if the
// request has not already set a user agent.
func NewHTTPTransport(logger Logger, dialer Dialer, tlsDialer TLSDialer) HTTPTransport {
	return WrapHTTPTransport(logger, NewOOHTTPBaseTransport(dialer, tlsDialer))
}

func NewOOHTTPBaseTransport(dialer Dialer, tlsDialer TLSDialer) HTTPTransport {
	// Using oohttp to support any TLS library.
	txp := oohttp.DefaultTransport.(*oohttp.Transport).Clone()

	// This wrapping ensures that we always have a timeout when we
	// are using HTTP; see https://github.com/ooni/probe/issues/1609.
	dialer = &httpDialerWithReadTimeout{dialer}
	txp.DialContext = dialer.DialContext
	tlsDialer = &httpTLSDialerWithReadTimeout{tlsDialer}
	txp.DialTLSContext = tlsDialer.DialTLSContext

	// We are using a different strategy to implement proxy: we
	// use a specific dialer that knows about proxying.
	txp.Proxy = nil

	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	txp.MaxConnsPerHost = 1

	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	txp.DisableCompression = true

	// Required to enable using HTTP/2 (which will be anyway forced
	// upon us when we are using TLS parroting).
	txp.ForceAttemptHTTP2 = true

	return &httpTransportConnectionsCloser{
		HTTPTransport: &oohttp.StdlibTransport{Transport: txp},
		Dialer:        dialer,
		TLSDialer:     tlsDialer,
	}
}

func WrapHTTPTransport(logger Logger, txp HTTPTransport) HTTPTransport {
	// Ensure we correctly forward CloseIdleConnections and compose
	// with a logging transport thus enabling logging.
	return &httpUserAgentTransport{
		HTTPTransport: &httpTransportLogger{
			HTTPTransport: txp,
			Logger:        logger,
		},
	}
}

// httpDialerWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpDialerWithReadTimeout struct {
	Dialer
}

// DialContext implements Dialer.DialContext.
func (d *httpDialerWithReadTimeout) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &httpConnWithReadTimeout{conn}, nil
}

// httpTLSDialerWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpTLSDialerWithReadTimeout struct {
	TLSDialer
}

// ErrNotTLSConn indicates that a TLSDialer returns a net.Conn
// that does not implement the TLSConn interface. This error should
// only happen when we do something wrong setting up HTTP code.
var ErrNotTLSConn = errors.New("not a TLSConn")

// DialTLSContext implements TLSDialer's DialTLSContext.
func (d *httpTLSDialerWithReadTimeout) DialTLSContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.TLSDialer.DialTLSContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	tconn, okay := conn.(TLSConn) // part of the contract but let's be graceful
	if !okay {
		conn.Close() // we own the conn here
		return nil, ErrNotTLSConn
	}
	return &httpTLSConnWithReadTimeout{tconn}, nil
}

// httpConnWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpConnWithReadTimeout struct {
	net.Conn
}

// httpConnReadTimeout is the read timeout we apply to all HTTP
// conns (see https://github.com/ooni/probe/issues/1609).
//
// This timeout is meant as a fallback mechanism so that a stuck
// connection will _eventually_ fail. This is why it is set to
// a large value (300 seconds when writing this note).
//
// There should be other mechanisms to ensure that the code is
// lively: the context during the RoundTrip and iox.ReadAllContext
// when reading the body. They should kick in earlier. But we
// additionally want to avoid leaking a (parked?) connection and
// the corresponding goroutine, hence this large timeout.
//
// A future @bassosimone may understand this problem even better
// and possibly apply an even better fix to this issue. This
// will happen when we'll be able to further study the anomalies
// described in https://github.com/ooni/probe/issues/1609.
const httpConnReadTimeout = 300 * time.Second

// Read implements Conn.Read.
func (c *httpConnWithReadTimeout) Read(b []byte) (int, error) {
	c.Conn.SetReadDeadline(time.Now().Add(httpConnReadTimeout))
	defer c.Conn.SetReadDeadline(time.Time{})
	return c.Conn.Read(b)
}

// httpTLSConnWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpTLSConnWithReadTimeout struct {
	TLSConn
}

// Read implements Conn.Read.
func (c *httpTLSConnWithReadTimeout) Read(b []byte) (int, error) {
	c.TLSConn.SetReadDeadline(time.Now().Add(httpConnReadTimeout))
	defer c.TLSConn.SetReadDeadline(time.Time{})
	return c.TLSConn.Read(b)
}

// httpUserAgentTransport is a transport that ensures that we always
// set an OONI specific default User-Agent header.
type httpUserAgentTransport struct {
	HTTPTransport
}

const defaultHTTPUserAgent = "miniooni/0.1.0-dev"

func (txp *httpUserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultHTTPUserAgent)
	}
	return txp.HTTPTransport.RoundTrip(req)
}

var _ HTTPTransport = &httpUserAgentTransport{}

// NewHTTPTransportStdlib creates a new HTTPTransport that uses
// the Go standard library for all operations, including DNS
// resolutions and TLS handshakes.
func NewHTTPTransportStdlib(logger Logger) HTTPTransport {
	dialer := NewDialerWithResolver(logger, NewResolverStdlib(logger))
	tlsDialer := NewTLSDialer(dialer, NewTLSHandshakerStdlib(logger))
	return NewHTTPTransport(logger, dialer, tlsDialer)
}
