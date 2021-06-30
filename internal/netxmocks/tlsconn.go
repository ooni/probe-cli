package netxmocks

import "crypto/tls"

// TLSConn allows to mock netxlite.TLSConn.
type TLSConn struct {
	// Conn is the embedded mockable Conn.
	Conn

	// MockConnectionState allows to mock the ConnectionState method.
	MockConnectionState func() tls.ConnectionState

	// MockHandshake allows to mock the Handshake method.
	MockHandshake func() error
}

// ConnectionState calls MockConnectionState.
func (c *TLSConn) ConnectionState() tls.ConnectionState {
	return c.MockConnectionState()
}

// Handshake calls MockHandshake.
func (c *TLSConn) Handshake() error {
	return c.MockHandshake()
}
