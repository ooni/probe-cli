package netplumbing

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/bassosimone/quic-go"
	"github.com/bassosimone/quic-go/http3"
)

// Transport implements Transport.
type Transport struct {
	// RoundTripper is the underlying http.Transport. You need to
	// configure this field. Otherwise, use NewTransport to obtain
	// a default configured Transport.
	RoundTripper *http.Transport

	// HTTP3RoundTripper is the underlying http3.Transport. You need
	// to configure this field. Otherwise, use NewTransport to obtain
	// a default configured Transport.
	HTTP3RoundTripper *http3.RoundTripper
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
	txp.HTTP3RoundTripper = &http3.RoundTripper{
		DisableCompression: true,
		Dial: func(ctx context.Context, network string, address string,
			tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
			return txp.QUICDialContext(ctx, network, address, tlsConfig, quicConfig)
		},
	}
	return txp
}

// DefaultTransport is the default implementation of Transport.
var DefaultTransport = NewTransport()
