package httptransport

import "net/http"

// UserAgentTransport is a transport that ensures that we always
// set an OONI specific default User-Agent header.
type UserAgentTransport struct {
	RoundTripper
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp UserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "miniooni/0.1.0-dev")
	}
	return txp.RoundTripper.RoundTrip(req)
}

var _ RoundTripper = UserAgentTransport{}
