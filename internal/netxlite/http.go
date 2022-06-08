package netxlite

//
// HTTP/1.1 and HTTP2 code
//

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// httpTransportErrWrapper is an HTTPTransport with error wrapping.
type httpTransportErrWrapper struct {
	HTTPTransport model.HTTPTransport
}

var _ model.HTTPTransport = &httpTransportErrWrapper{}

func (txp *httpTransportErrWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := txp.HTTPTransport.RoundTrip(req)
	if err != nil {
		return nil, NewTopLevelGenericErrWrapper(err)
	}
	return resp, nil
}

func (txp *httpTransportErrWrapper) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
}

func (txp *httpTransportErrWrapper) Network() string {
	return txp.HTTPTransport.Network()
}

// httpTransportLogger is an HTTPTransport with logging.
type httpTransportLogger struct {
	// HTTPTransport is the underlying HTTP transport.
	HTTPTransport model.HTTPTransport

	// Logger is the underlying logger.
	Logger model.DebugLogger
}

var _ model.HTTPTransport = &httpTransportLogger{}

func (txp *httpTransportLogger) RoundTrip(req *http.Request) (*http.Response, error) {
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

func (txp *httpTransportLogger) Network() string {
	return txp.HTTPTransport.Network()
}

// httpTransportConnectionsCloser is an HTTPTransport that
// correctly forwards CloseIdleConnections calls.
type httpTransportConnectionsCloser struct {
	HTTPTransport model.HTTPTransport
	Dialer        model.Dialer
	TLSDialer     model.TLSDialer
}

var _ model.HTTPTransport = &httpTransportConnectionsCloser{}

func (txp *httpTransportConnectionsCloser) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.HTTPTransport.RoundTrip(req)
}

func (txp *httpTransportConnectionsCloser) Network() string {
	return txp.HTTPTransport.Network()
}

// CloseIdleConnections forwards the CloseIdleConnections calls.
func (txp *httpTransportConnectionsCloser) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
	txp.Dialer.CloseIdleConnections()
	txp.TLSDialer.CloseIdleConnections()
}

// NewHTTPTransportWithResolver creates a new HTTP transport using
// the stdlib for everything but the given resolver.
func NewHTTPTransportWithResolver(logger model.DebugLogger, reso model.Resolver) model.HTTPTransport {
	dialer := NewDialerWithResolver(logger, reso)
	thx := NewTLSHandshakerStdlib(logger)
	tlsDialer := NewTLSDialer(dialer, thx)
	return NewHTTPTransport(logger, dialer, tlsDialer)
}

// NewHTTPTransport returns a wrapped HTTP transport for HTTP2 and HTTP/1.1
// using the given dialer and logger.
//
// The returned transport will gracefully handle TLS connections
// created using gitlab.com/yawning/utls.git, if the TLS dialer
// is a dialer using such library for TLS operations.
//
// The returned transport will not have a configured proxy, not
// even the proxy configurable from the environment.
//
// QUIRK: the returned transport will disable transparent decompression
// of compressed response bodies (and will not automatically
// ask for such compression, though you can always do that manually).
//
// The returned transport will configure TCP and TLS connections
// created using its dialer and TLS dialer to always have a
// read watchdog timeout to address https://github.com/ooni/probe/issues/1609.
//
// QUIRK: the returned transport will always enforce 1 connection per host
// and we cannot get rid of this QUIRK requirement because it is
// necessary to perform sane measurements with tracing. We will be
// able to possibly relax this requirement after we change the
// way in which we perform measurements.
//
// This factory and NewHTTPTransportStdlib are the recommended
// ways of creating a new HTTPTransport.
func NewHTTPTransport(logger model.DebugLogger, dialer model.Dialer, tlsDialer model.TLSDialer) model.HTTPTransport {
	return WrapHTTPTransport(logger, newOOHTTPBaseTransport(dialer, tlsDialer))
}

// newOOHTTPBaseTransport is the low-level factory used by NewHTTPTransport
// to create a new, suitable HTTPTransport for HTTP2 and HTTP/1.1.
//
// This factory uses github.com/ooni/oohttp, hence its name.
func newOOHTTPBaseTransport(dialer model.Dialer, tlsDialer model.TLSDialer) model.HTTPTransport {
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

	// Ensure we correctly forward CloseIdleConnections.
	return &httpTransportConnectionsCloser{
		HTTPTransport: &httpTransportStdlib{&oohttp.StdlibTransport{Transport: txp}},
		Dialer:        dialer,
		TLSDialer:     tlsDialer,
	}
}

// stdlibTransport wraps oohttp.StdlibTransport to add .Network()
type httpTransportStdlib struct {
	StdlibTransport *oohttp.StdlibTransport
}

var _ model.HTTPTransport = &httpTransportStdlib{}

func (txp *httpTransportStdlib) CloseIdleConnections() {
	txp.StdlibTransport.CloseIdleConnections()
}

func (txp *httpTransportStdlib) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.StdlibTransport.RoundTrip(req)
}

// Network implements HTTPTransport.Network.
func (txp *httpTransportStdlib) Network() string {
	return "tcp"
}

// WrapHTTPTransport creates an HTTPTransport using the given logger
// and guarantees that returned errors are wrapped.
//
// This is a low level factory. Consider not using it directly.
func WrapHTTPTransport(logger model.DebugLogger, txp model.HTTPTransport) model.HTTPTransport {
	return &httpTransportLogger{
		HTTPTransport: &httpTransportErrWrapper{txp},
		Logger:        logger,
	}
}

// httpDialerWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpDialerWithReadTimeout struct {
	Dialer model.Dialer
}

var _ model.Dialer = &httpDialerWithReadTimeout{}

func (d *httpDialerWithReadTimeout) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
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
	TLSDialer model.TLSDialer
}

var _ model.TLSDialer = &httpTLSDialerWithReadTimeout{}

func (d *httpTLSDialerWithReadTimeout) CloseIdleConnections() {
	d.TLSDialer.CloseIdleConnections()
}

// ErrNotTLSConn occur when an interface accepts a net.Conn but
// internally needs a TLSConn and you pass a net.Conn that doesn't
// implement TLSConn to such an interface.
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

// NewHTTPTransportStdlib creates a new HTTPTransport using
// the stdlib for DNS resolutions and TLS.
//
// This factory calls NewHTTPTransport with suitable dialers.
//
// This factory and NewHTTPTransport are the recommended
// ways of creating a new HTTPTransport.
func NewHTTPTransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	dialer := NewDialerWithResolver(logger, NewStdlibResolver(logger))
	tlsDialer := NewTLSDialer(dialer, NewTLSHandshakerStdlib(logger))
	return NewHTTPTransport(logger, dialer, tlsDialer)
}

// NewHTTPClientStdlib creates a new HTTPClient that uses the
// standard library for TLS and DNS resolutions.
func NewHTTPClientStdlib(logger model.DebugLogger) model.HTTPClient {
	txp := NewHTTPTransportStdlib(logger)
	return NewHTTPClient(txp)
}

// NewHTTPClient creates a new, wrapped HTTPClient using the given transport.
func NewHTTPClient(txp model.HTTPTransport) model.HTTPClient {
	return WrapHTTPClient(&http.Client{Transport: txp})
}

// WrapHTTPClient wraps an HTTP client to add error wrapping capabilities.
func WrapHTTPClient(clnt model.HTTPClient) model.HTTPClient {
	return &httpClientErrWrapper{clnt}
}

type httpClientErrWrapper struct {
	HTTPClient model.HTTPClient
}

func (c *httpClientErrWrapper) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, NewTopLevelGenericErrWrapper(err)
	}
	return resp, nil
}

func (c *httpClientErrWrapper) CloseIdleConnections() {
	c.HTTPClient.CloseIdleConnections()
}
