package netxlite

//
// HTTP3 code
//

import (
	"crypto/tls"
	"io"
	"net/http"

	"github.com/lucas-clemente/quic-go/http3"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// http3RoundTripper is the abstract type of quic-go/http3.RoundTripper.
type http3RoundTripper interface {
	http.RoundTripper
	io.Closer
}

// http3Transport is an HTTPTransport using the http3 protocol.
type http3Transport struct {
	child  http3RoundTripper
	dialer model.QUICDialer
}

var _ model.HTTPTransport = &http3Transport{}

// Network implements HTTPTransport.Network.
func (txp *http3Transport) Network() string {
	return "quic"
}

// RoundTrip implements HTTPTransport.RoundTrip.
func (txp *http3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.child.RoundTrip(req)
}

// CloseIdleConnections implements HTTPTransport.CloseIdleConnections.
func (txp *http3Transport) CloseIdleConnections() {
	txp.child.Close()
	txp.dialer.CloseIdleConnections()
}

// NewHTTP3Transport creates a new HTTPTransport using http3. The
// dialer argument MUST NOT be nil. If the tlsConfig argument is nil,
// then the code will use the default TLS configuration.
func NewHTTP3Transport(
	logger model.DebugLogger, dialer model.QUICDialer, tlsConfig *tls.Config) model.HTTPTransport {
	return WrapHTTPTransport(logger, &http3Transport{
		child: &http3.RoundTripper{
			Dial: dialer.DialContext,
			// The following (1) reduces the number of headers that Go will
			// automatically send for us and (2) ensures that we always receive
			// back the true headers, such as Content-Length. This change is
			// functional to OONI's goal of observing the network.
			DisableCompression: true,
			TLSClientConfig:    tlsConfig,
		},
		dialer: dialer,
	})
}

// NewHTTP3TransportStdlib creates a new HTTPTransport using http3 that
// uses standard functionality for everything but the logger.
func NewHTTP3TransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	ql := NewQUICListener()
	reso := NewStdlibResolver(logger)
	qd := NewQUICDialerWithResolver(ql, logger, reso)
	return NewHTTP3Transport(logger, qd, nil)
}

// NewHTTPTransportWithResolver creates a new HTTPTransport using http3
// that uses the given logger and the given resolver.
func NewHTTP3TransportWithResolver(logger model.Logger, reso model.Resolver) model.HTTPTransport {
	qd := NewQUICDialerWithResolver(NewQUICListener(), logger, reso)
	return NewHTTP3Transport(logger, qd, nil)
}
