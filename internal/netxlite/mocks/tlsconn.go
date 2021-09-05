package mocks

import (
	"context"
	"crypto/tls"
)

// TLSConn allows to mock netxlite.TLSConn.
type TLSConn struct {
	// Conn is the embedded mockable Conn.
	Conn

	// MockConnectionState allows to mock the ConnectionState method.
	MockConnectionState func() tls.ConnectionState

	// MockHandshakeContext allows to mock the HandshakeContext method.
	MockHandshakeContext func(ctx context.Context) error
}

// ConnectionState calls MockConnectionState.
func (c *TLSConn) ConnectionState() tls.ConnectionState {
	return c.MockConnectionState()
}

// HandshakeContext calls MockHandshakeContext.
func (c *TLSConn) HandshakeContext(ctx context.Context) error {
	return c.MockHandshakeContext(ctx)
}
