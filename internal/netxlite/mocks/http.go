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

// HTTPClient allows mocking an http.Client.
type HTTPClient struct {
	MockDo func(req *http.Request) (*http.Response, error)

	MockCloseIdleConnections func()
}

// Do calls MockDo.
func (txp *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return txp.MockDo(req)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (txp *HTTPClient) CloseIdleConnections() {
	txp.MockCloseIdleConnections()
}
