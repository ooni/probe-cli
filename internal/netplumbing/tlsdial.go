package netplumbing

import (
	"context"
	"crypto/tls"
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

// DialTLSContext dials a TLS connection.
func (txp *Transport) DialTLSContext(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	if settings := ContextSettings(ctx); settings != nil && settings.Proxy != nil {
		return nil, &ErrDialTLS{ErrProxyNotImplemented}
	}
	conn, err := txp.directDialTLSContext(ctx, network, addr)
	if err != nil {
		return nil, &ErrDialTLS{err}
	}
	return conn, nil
}

// directDialTLSContext is a dialTLSContext that does not use a proxy.
func (txp *Transport) directDialTLSContext(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	log := txp.logger(ctx)
	log.Debugf("dialTLS: %s/%s...", addr, network)
	conn, err := txp.doDialTLSContext(ctx, network, addr)
	if err != nil {
		log.Debugf("dialTLS: %s/%s... %s", addr, network, err)
		return nil, err
	}
	log.Debugf("dialTLS: %s/%s... ok", addr, network)
	return conn, nil
}

// tlsClientConfig returns the configured TLS client config or the default.
func (txp *Transport) tlsClientConfig(ctx context.Context) *tls.Config {
	if settings := ContextSettings(ctx); settings != nil && settings.TLSClientConfig != nil {
		return settings.TLSClientConfig.Clone()
	}
	return &tls.Config{}
}

// tlsHandshakeTimeout returns the TLS handshake timeout.
func (txp *Transport) tlsHandshakeTimeout() time.Duration {
	return 10 * time.Second
}

// doDialTLSContext implements dialTLSContext.
func (txp *Transport) doDialTLSContext(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	sni, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	tcpConn, err := txp.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	tlsConfig := txp.tlsClientConfig(ctx)
	// TODO(bassosimone): implement this part
	//if tlsConfig.RootCAs == nil {
	//}
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
	tcpConn.SetDeadline(time.Now().Add(txp.tlsHandshakeTimeout()))
	tlsConn, _, err := txp.TLSHandshake(ctx, tcpConn, tlsConfig)
	if err != nil {
		tcpConn.Close() // we own the connection
		return nil, err
	}
	tcpConn.SetDeadline(time.Time{})
	return tlsConn, nil
}
