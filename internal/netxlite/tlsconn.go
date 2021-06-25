package netxlite

import (
	"crypto/tls"
	"net"
)

// TLSConn is any tls.Conn-like structure.
type TLSConn interface {
	// net.Conn is the embedded conn.
	net.Conn

	// ConnectionState returns the TLS connection state.
	ConnectionState() tls.ConnectionState

	// Handshake performs the handshake.
	Handshake() error
}
