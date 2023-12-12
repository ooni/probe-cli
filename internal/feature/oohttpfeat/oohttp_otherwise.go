//go:build go1.21 || ooni_feature_disable_oohttp

package oohttpfeat

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
)

// HTTPTransport is a wrapper for either net/http or oohttp's Transport.
type HTTPTransport struct {
	txp *http.Transport
}

// HTTPRequest is the type of the underlying *http.Request we're using in this library, which
// in turns depends on how we're being compiled.
type HTTPRequest = http.Request

// ExpectedForceAttemptHTTP2 is the expected value returned by GetForceAttemptHTTP2.
const ExpectedForceAttemptHTTP2 = false

// NewHTTPTransport creates a new [*HTTPTransport] instance.
func NewHTTPTransport() *HTTPTransport {
	txp := &HTTPTransport{http.DefaultTransport.(*http.Transport).Clone()}

	// When we're using our net/http fork, we know we're able to
	// actually force HTTP/2 without any issue
	txp.txp.ForceAttemptHTTP2 = ExpectedForceAttemptHTTP2

	return txp
}

// SetDialContext sets the DialContext field.
func (txp *HTTPTransport) SetDialContext(fx func(ctx context.Context, network, address string) (net.Conn, error)) {
	txp.txp.DialContext = fx
}

// GetDialContext returns the value of the DialContext field.
func (txp *HTTPTransport) GetDialContext() func(ctx context.Context, network, address string) (net.Conn, error) {
	return txp.txp.DialContext
}

// SetDialTLSContext sets the DialTLSContext field.
func (txp *HTTPTransport) SetDialTLSContext(fx func(ctx context.Context, network, address string) (net.Conn, error)) {
	txp.txp.DialTLSContext = fx
}

// GetDialTLSContext returns the value of the DialTLSContext field.
func (txp *HTTPTransport) GetDialTLSContext() func(ctx context.Context, network, address string) (net.Conn, error) {
	return txp.txp.DialTLSContext
}

// SetProxy sets the Proxy field.
func (txp *HTTPTransport) SetProxy(fx func(*HTTPRequest) (*url.URL, error)) {
	txp.txp.Proxy = fx
}

// GetProxy returns the value of the Proxy field.
func (txp *HTTPTransport) GetProxy() func(*HTTPRequest) (*url.URL, error) {
	return txp.txp.Proxy
}

// SetMaxConnsPerHost sets the MaxConnsPerHost field.
func (txp *HTTPTransport) SetMaxConnsPerHost(value int) {
	txp.txp.MaxConnsPerHost = value
}

// GetMaxConnsPerHost returns the value of the MaxConnsPerHost field.
func (txp *HTTPTransport) GetMaxConnsPerHost() int {
	return txp.txp.MaxConnsPerHost
}

// SetDisableCompression sets the DisableCompression field.
func (txp *HTTPTransport) SetDisableCompression(value bool) {
	txp.txp.DisableCompression = value
}

// GetDisableCompression returns the value of the DisableCompression field.
func (txp *HTTPTransport) GetDisableCompression() bool {
	return txp.txp.DisableCompression
}

// SetTLSClientConfig sets the TLSClientConfig field.
func (txp *HTTPTransport) SetTLSClientConfig(value *tls.Config) {
	txp.txp.TLSClientConfig = value
}

// GetForceAttemptHTTP2 returns the value of the ForceAttemptHTTP2 field.
func (txp *HTTPTransport) GetForceAttemptHTTP2() bool {
	return txp.txp.ForceAttemptHTTP2
}

// CloseIdleConnections closes the idle connections.
func (txp *HTTPTransport) CloseIdleConnections() {
	txp.txp.CloseIdleConnections()
}

// RoundTrip performs an HTTP round trip.
func (txp *HTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.txp.RoundTrip(req)
}
