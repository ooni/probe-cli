package mocks

import "net/http"

// HTTPTransport mocks netxlite.HTTPTransport.
type HTTPTransport struct {
	MockRoundTrip            func(req *http.Request) (*http.Response, error)
	MockCloseIdleConnections func()
}

// RoundTrip calls MockRoundTrip.
func (txp *HTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.MockRoundTrip(req)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (txp *HTTPTransport) CloseIdleConnections() {
	txp.MockCloseIdleConnections()
}
