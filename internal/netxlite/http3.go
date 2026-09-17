package netxlite

//
// HTTP3 code
//

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
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
	return "udp"
}

// RoundTrip implements HTTPTransport.RoundTrip.
func (txp *http3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.child.RoundTrip(req)
}

// CloseIdleConnections implements HTTPTransport.CloseIdleConnections.
func (txp *http3Transport) CloseIdleConnections() {
	_ = txp.child.Close()
	txp.dialer.CloseIdleConnections()
}

// quicConnForHTTP3 unwraps a [model.QUICConn] to the concrete *quic.Conn required
// by http3.Transport.Dial.
func quicConnForHTTP3(qconn model.QUICConn) *quic.Conn {
	if u, ok := qconn.(quicConnUnwrapper); ok {
		return u.unwrapForHTTP3()
	}
	return qconn.(*quic.Conn)
}

// NewHTTP3Transport creates a new HTTPTransport using http3. The
// dialer argument MUST NOT be nil. If the tlsConfig argument is nil,
// then the code will use the default TLS configuration.
func NewHTTP3Transport(
	logger model.DebugLogger, dialer model.QUICDialer, tlsConfig *tls.Config) model.HTTPTransport {
	return WrapHTTPTransport(logger, &http3Transport{
		child: &http3.Transport{
			Dial: func(ctx context.Context, addr string,
				tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
				qconn, err := dialer.DialContext(ctx, addr, tlsCfg, cfg)
				if err != nil {
					return nil, err
				}
				return quicConnForHTTP3(qconn), nil
			},
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
func (netx *Netx) NewHTTP3TransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	ql := netx.NewUDPListener()
	reso := netx.NewStdlibResolver(logger)
	qd := netx.NewQUICDialerWithResolver(ql, logger, reso)
	return NewHTTP3Transport(logger, qd, nil)
}

// NewHTTPTransportWithResolver creates a new HTTPTransport using http3
// that uses the given logger and the given resolver.
func NewHTTP3TransportWithResolver(netx *Netx, logger model.DebugLogger, reso model.Resolver) model.HTTPTransport {
	qd := netx.NewQUICDialerWithResolver(netx.NewUDPListener(), logger, reso)
	return NewHTTP3Transport(logger, qd, nil)
}

// NewHTTP3ClientWithResolver creates a new HTTP3Transport using the
// given resolver and then from that builds an HTTPClient.
func NewHTTP3ClientWithResolver(netx *Netx, logger model.Logger, reso model.Resolver) model.HTTPClient {
	return NewHTTPClient(NewHTTP3TransportWithResolver(netx, logger, reso))
}
