package oldhttptransport

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/transactionid"
)

// Transactioner performs single HTTP transactions.
type Transactioner struct {
	roundTripper http.RoundTripper
}

// NewTransactioner creates a new Transport.
func NewTransactioner(roundTripper http.RoundTripper) *Transactioner {
	return &Transactioner{
		roundTripper: roundTripper,
	}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *Transactioner) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.roundTripper.RoundTrip(req.WithContext(
		transactionid.WithTransactionID(req.Context()),
	))
}

// CloseIdleConnections closes the idle connections.
func (t *Transactioner) CloseIdleConnections() {
	// Adapted from net/http code
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := t.roundTripper.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}
