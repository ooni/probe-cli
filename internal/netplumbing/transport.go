package netplumbing

import (
	"net/http"
	"time"
)

// Transport implements Transport.
type Transport struct {
	// RoundTripper is the underlying http.Transport. You need to
	// configure this field. Otherwise, use NewTransport to obtain
	// a default configured Transport.
	RoundTripper *http.Transport
}

// NewTransport creates a new instance of Transport using a
// default underlying http.Transport.
func NewTransport() *Transport {
	txp := &Transport{}
	txp.RoundTripper = &http.Transport{
		Proxy:                 txp.proxy,
		DialContext:           txp.directDialContext,
		DialTLSContext:        txp.DialTLSContext,
		TLSHandshakeTimeout:   txp.tlsHandshakeTimeout(),
		DisableCompression:    true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	return txp
}

// DefaultTransport is the default implementation of Transport.
var DefaultTransport = NewTransport()
