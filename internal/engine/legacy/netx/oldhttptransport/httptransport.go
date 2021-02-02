// Package oldhttptransport contains HTTP transport extensions. Here we
// define a http.Transport that emits events.
package oldhttptransport

import (
	"net/http"
)

// Transport performs single HTTP transactions and emits
// measurement events as they happen.
type Transport struct {
	roundTripper http.RoundTripper
}

// New creates a new Transport.
func New(roundTripper http.RoundTripper) *Transport {
	return &Transport{
		roundTripper: NewTransactioner(NewBodyTracer(
			NewTraceTripper(roundTripper))),
	}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	// Make sure we're not sending Go's default User-Agent
	// if the user has configured no user agent
	if req.Header.Get("User-Agent") == "" {
		req.Header["User-Agent"] = nil
	}
	return t.roundTripper.RoundTrip(req)
}

// CloseIdleConnections closes the idle connections.
func (t *Transport) CloseIdleConnections() {
	// Adapted from net/http code
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := t.roundTripper.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}
