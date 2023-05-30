package mocks

import (
	"context"
	"crypto/tls"
	"net"
)

// TLSHandshaker is a mockable TLS handshaker.
type TLSHandshaker struct {
	MockHandshake func(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}

// Handshake calls MockHandshake.
func (th *TLSHandshaker) Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
	net.Conn, tls.ConnectionState, error) {
	return th.MockHandshake(ctx, conn, config)
}

// TLSConn allows to mock netxlite.TLSConn.
type TLSConn struct {
	// Conn is the embedded mockable Conn.
	Conn

	// MockConnectionState allows to mock the ConnectionState method.
	MockConnectionState func() tls.ConnectionState

	// MockHandshakeContext allows to mock the HandshakeContext method.
	MockHandshakeContext func(ctx context.Context) error

	// MockNetConn returns the underlying net.Conn
	MockNetConn func() net.Conn
}

// ConnectionState calls MockConnectionState.
func (c *TLSConn) ConnectionState() tls.ConnectionState {
	return c.MockConnectionState()
}

// HandshakeContext calls MockHandshakeContext.
func (c *TLSConn) HandshakeContext(ctx context.Context) error {
	return c.MockHandshakeContext(ctx)
}

// NetConn calls MockNetConn.
func (c *TLSConn) NetConn() net.Conn {
	return c.MockNetConn()
}

// TLSDialer allows to mock netxlite.TLSDialer.
type TLSDialer struct {
	// MockCloseIdleConnections allows to mock the CloseIdleConnections method.
	MockCloseIdleConnections func()

	// MockDialTLSContext allows to mock the DialTLSContext method.
	MockDialTLSContext func(ctx context.Context, network, address string) (net.Conn, error)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (d *TLSDialer) CloseIdleConnections() {
	d.MockCloseIdleConnections()
}

// DialTLSContext calls MockDialTLSContext.
func (d *TLSDialer) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.MockDialTLSContext(ctx, network, address)
}
