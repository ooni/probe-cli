package netxlite

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
)

// http3Dialer adapts a QUICContextDialer to work with
// an http3.RoundTripper. This is necessary because the
// http3.RoundTripper does not support DialContext.
type http3Dialer struct {
	Dialer QUICContextDialer
}

// dial is like QUICContextDialer.DialContext but without context.
func (d *http3Dialer) dial(network, address string, tlsConfig *tls.Config,
	quicConfig *quic.Config) (quic.EarlySession, error) {
	return d.Dialer.DialContext(
		context.Background(), network, address, tlsConfig, quicConfig)
}

// http3Transport is an HTTPTransport using the http3 protocol.
type http3Transport struct {
	child *http3.RoundTripper
}

var _ HTTPTransport = &http3Transport{}

// RoundTrip implements HTTPTransport.RoundTrip.
func (txp *http3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.child.RoundTrip(req)
}

// CloseIdleConnections implements HTTPTransport.CloseIdleConnections.
func (txp *http3Transport) CloseIdleConnections() {
	txp.child.Close()
}

// NewHTTP3Transport creates a new HTTPTransport using http3. The
// dialer argument MUST NOT be nil. If the tlsConfig argument is nil,
// then the code will use the default TLS configuration.
func NewHTTP3Transport(
	dialer QUICContextDialer, tlsConfig *tls.Config) HTTPTransport {
	return &http3Transport{
		child: &http3.RoundTripper{
			Dial:            (&http3Dialer{dialer}).dial,
			TLSClientConfig: tlsConfig,
		},
	}
}
