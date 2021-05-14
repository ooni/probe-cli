package oonet

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

// ErrDialTLS is an error when dialing a TLS connection.
type ErrDialTLS struct {
	error
}

// Unwrap returns the wrapped error.
func (err *ErrDialTLS) Unwrap() error {
	return err.error
}

// DialTLSContext dials a TLS connection. On error, this function will always return
// an instance of ErrDialTLS. This function will use the configured Transport.Logger to
// log messages, if configured. This function will either use txp.DefaultDialTLSContext, by
// default, or ContextOverrides.DialTLSContext, if configured. This function will call
// txp.DialContext to dial a cleartext connection and txp.TLSHandshake to handshake.
func (txp *Transport) DialTLSContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	if txp.Proxy != nil {
		return nil, &ErrDialTLS{ErrProxyNotImplemented}
	}
	return txp.httpDialTLSContext(ctx, network, addr)
}

// httpDialTLSContext allows HTTP code to bypass the check on Transport.Proxy
// implemented by Transpoirt.DialTLSContext.
func (txp *Transport) httpDialTLSContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	log := txp.logger()
	log.Debugf("dialTLS: %s/%s...", addr, network)
	conn, err := txp.routeDialTLSContext(ctx, network, addr)
	if err != nil {
		log.Debugf("dialTLS: %s/%s... %s", addr, network, err)
		return nil, &ErrDialTLS{err}
	}
	log.Debugf("dialTLS: %s/%s... ok", addr, network)
	return conn, nil
}

// routeDialTLSContext routes the DialTLSContext call.
func (txp *Transport) routeDialTLSContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	if overrides := ContextOverrides(ctx); overrides != nil && overrides.DialTLSContext != nil {
		return overrides.DialTLSContext(ctx, network, addr)
	}
	return txp.DefaultDialTLSContext(ctx, network, addr)
}

// DefaultDialTLSContext is the default implementation of DialTLSContext.
func (txp *Transport) DefaultDialTLSContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	sni, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	conn, err := txp.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	tlsConfig := txp.tlsClientConfig()
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = sni
	}
	if tlsConfig.NextProtos == nil && port == "443" {
		tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	}
	// Set the deadline so the handshake fails naturally for I/O timeout
	// rather than for a context timeout. The context may still fail, when
	// the user wants that. So, we can distinguish the case where there
	// is a timeout from the impatient-user case.
	conn.SetDeadline(time.Now().Add(txp.tlsHandshakeTimeout()))
	result, err := txp.TLSHandshake(ctx, conn, tlsConfig)
	if err != nil {
		conn.Close() // we own the connection
		return nil, err
	}
	conn.SetDeadline(time.Time{})
	return result.Conn, nil
}

// ErrTLSHandshake is an error during the TLS handshake.
type ErrTLSHandshake struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrTLSHandshake) Unwrap() error {
	return err.error
}

// ErrTLSNoMutualProtocol indicates that there is no mutual TLS protocol.
type ErrTLSNoMutualProtocol struct {
	// Config contains the TLS config.
	Config *tls.Config

	// State contains the TLS connection state.
	State *tls.ConnectionState
}

// Error implements error.Error.
func (err *ErrTLSNoMutualProtocol) Error() string {
	return "oonet: cannot negotiate application-level protocol"
}

// TLSHandshake performs the TLS handshake. This function WILL NOT take ownership
// of the `conn` net.Conn parameter. This function will log using txp.Logger, if
// configured. By default, this function uses txp.DefaultTLSHandshake, but you can
// override this by setting ContextOverrides().TLSHandshake.
func (txp *Transport) TLSHandshake(ctx context.Context, conn net.Conn, config *tls.Config) (*TLSHandshakeResult, error) {
	log := txp.logger()
	prefix := fmt.Sprintf("tlsHandshake: %s/%s sni=%s alpn=%s...", conn.RemoteAddr().String(),
		conn.RemoteAddr().Network(), config.ServerName, config.NextProtos)
	log.Debug(prefix)
	result, err := txp.routeTLSHandshake(ctx, conn, config)
	if err != nil {
		log.Debugf("%s %s", prefix, err)
		return nil, &ErrTLSHandshake{err}
	}
	state := result.State
	// TODO(bassosimone): remove the following now-redundant check
	if len(config.NextProtos) > 0 && !state.NegotiatedProtocolIsMutual {
		err := &ErrTLSNoMutualProtocol{Config: config, State: state}
		log.Debugf("%s %s", prefix, err)
		return nil, err
	}
	log.Debugf("%s proto=%s", prefix, result.State.NegotiatedProtocol)
	return result, nil
}

// routeTLSHandshake routes the TLS handshake call.
func (txp *Transport) routeTLSHandshake(ctx context.Context, conn net.Conn, config *tls.Config) (*TLSHandshakeResult, error) {
	if overrides := ContextOverrides(ctx); overrides != nil && overrides.TLSHandshake != nil {
		return overrides.TLSHandshake(ctx, conn, config)
	}
	return txp.DefaultTLSHandshake(ctx, conn, config)
}

// DefaultTLSHandshake is the default TLSHandshake implementation.
func (txp *Transport) DefaultTLSHandshake(ctx context.Context, conn net.Conn, config *tls.Config) (*TLSHandshakeResult, error) {
	tlsConn := tls.Client(conn, config)
	errch := make(chan error, 1)
	go func() { errch <- tlsConn.Handshake() }()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errch:
		if err != nil {
			return nil, err
		}
		state := tlsConn.ConnectionState()
		return &TLSHandshakeResult{Conn: tlsConn, State: &state}, nil
	}
}
