package netplumbing

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
)

// TLSHandshaker performs a TLS handshake.
type TLSHandshaker interface {
	// TLSHandshake performs the TLS handshake.
	TLSHandshake(ctx context.Context, tcpConn net.Conn, config *tls.Config) (
		tlsConn net.Conn, state *tls.ConnectionState, err error)
}

// ErrTLSHandshake is an error during the TLS handshake.
type ErrTLSHandshake struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrTLSHandshake) Unwrap() error {
	return err.error
}

// TLSHandshake implements TLSHandshaker.Handshake.
func (txp *Transport) TLSHandshake(
	ctx context.Context, tcpConn net.Conn, config *tls.Config) (
	net.Conn, *tls.ConnectionState, error) {
	log := txp.logger(ctx)
	prefix := fmt.Sprintf("tlsHandshake: %s/%s sni=%s alpn=%s...", tcpConn.RemoteAddr().String(),
		tcpConn.RemoteAddr().Network(), config.ServerName, config.NextProtos)
	log.Debug(prefix)
	tlsConn, state, err := txp.routeTLSHandshake(ctx, tcpConn, config)
	if err != nil {
		log.Debugf("%s %s", prefix, err)
		return nil, nil, &ErrTLSHandshake{err}
	}
	log.Debugf("%s proto=%s", prefix, state.NegotiatedProtocol)
	return tlsConn, state, nil
}

// StdlibTLSHandshaker uses the stdlib to perform the TLS handshake.
type StdlibTLSHandshaker struct{}

// TLSHandshake implements TLSHandshaker.TLSHandshake.
func (th *StdlibTLSHandshaker) TLSHandshake(
	ctx context.Context, tcpConn net.Conn, config *tls.Config) (
	net.Conn, *tls.ConnectionState, error) {
	tlsConn := tls.Client(tcpConn, config)
	errch := make(chan error, 1)
	go func() { errch <- tlsConn.Handshake() }()
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case err := <-errch:
		if err != nil {
			return nil, nil, err
		}
		state := tlsConn.ConnectionState()
		return tlsConn, &state, nil
	}
}

// DefaultTLSHandshaker is the TLSHandshaker used by default by this transport.
func (txp *Transport) DefaultTLSHandshaker() TLSHandshaker {
	return &StdlibTLSHandshaker{}
}

// routeTLSHandshake routes the TLS handshake call.
func (txp *Transport) routeTLSHandshake(
	ctx context.Context, tcpConn net.Conn, tlsConfig *tls.Config) (
	net.Conn, *tls.ConnectionState, error) {
	if config := ContextConfig(ctx); config != nil && config.TLSHandshaker != nil {
		return config.TLSHandshaker.TLSHandshake(ctx, tcpConn, tlsConfig)
	}
	return txp.DefaultTLSHandshaker().TLSHandshake(ctx, tcpConn, tlsConfig)
}
