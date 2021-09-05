package netxlite

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
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

// NewHTTPTransport creates a new HTTP transport using Go stdlib.
func NewHTTPTransport(dialer Dialer, tlsConfig *tls.Config,
	handshaker TLSHandshaker) HTTPTransport {
	txp := http.DefaultTransport.(*http.Transport).Clone()
	dialer = &httpDialerWithReadTimeout{dialer}
	txp.DialContext = dialer.DialContext
	txp.DialTLSContext = (&TLSDialer{
		Config:        tlsConfig,
		Dialer:        dialer,
		TLSHandshaker: handshaker,
	}).DialTLSContext
	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	txp.MaxConnsPerHost = 1
	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	txp.DisableCompression = true
	return txp
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
