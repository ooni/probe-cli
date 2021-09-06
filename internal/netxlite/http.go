package netxlite

import (
	"context"
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

// RoundTrip implements HTTPTransport.RoundTrip.
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

// CloseIdleConnections implement HTTPTransport.CloseIdleConnections.
func (txp *httpTransportLogger) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
}

// httpTransportConnectionsCloser is an HTTPTransport that
// correctly forwards CloseIdleConnections.
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
// dialer and TLS handshaker to create connections.
//
// We need a TLS handshaker here, as opposed to a TLSDialer, because we
// wrap the dialer we'll use to enforce timeouts for HTTP idle
// connections (see https://github.com/ooni/probe/issues/1609 for more info).
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
func NewHTTPTransport(logger Logger, dialer Dialer, tlsHandshaker TLSHandshaker) HTTPTransport {
	// Using oohttp to support any TLS library.
	txp := oohttp.DefaultTransport.(*oohttp.Transport).Clone()

	// This wrapping ensures that we always have a timeout when we
	// are using HTTP; see https://github.com/ooni/probe/issues/1609.
	dialer = &httpDialerWithReadTimeout{dialer}
	txp.DialContext = dialer.DialContext
	tlsDialer := NewTLSDialer(dialer, tlsHandshaker)
	txp.DialTLSContext = tlsDialer.DialTLSContext

	// We are using a different strategy to implement proxy: we
	// use a specific dialer that knows about proxying.
	txp.Proxy = nil

	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	//
	// UNDOCUMENTED: I am wondering whether we can relax this constraint.
	txp.MaxConnsPerHost = 1

	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	txp.DisableCompression = true

	// Required to enable using HTTP/2 (which will be anyway forced
	// upon us when we are using TLS parroting).
	txp.ForceAttemptHTTP2 = true

	// Ensure we correctly forward CloseIdleConnections and compose
	// with a logging transport thus enabling logging.
	return &httpTransportLogger{
		HTTPTransport: &httpTransportConnectionsCloser{
			HTTPTransport: &oohttp.StdlibTransport{Transport: txp},
			Dialer:        dialer,
			TLSDialer:     tlsDialer,
		},
		Logger: logger,
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

// httpConnWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpConnWithReadTimeout struct {
	net.Conn
}

// Read implements Conn.Read.
func (c *httpConnWithReadTimeout) Read(b []byte) (int, error) {
	c.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetReadDeadline(time.Time{})
	return c.Conn.Read(b)
}
