package mocks

import "net/http"

// HTTP3RoundTripper allows mocking http3.RoundTripper.
type HTTP3RoundTripper struct {
	MockRoundTrip func(req *http.Request) (*http.Response, error)
	MockClose     func() error
}

// RoundTrip calls MockRoundTrip.
func (txp *HTTP3RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.MockRoundTrip(req)
}

// Close calls MockClose.
func (txp *HTTP3RoundTripper) Close() error {
	return txp.MockClose()
}
