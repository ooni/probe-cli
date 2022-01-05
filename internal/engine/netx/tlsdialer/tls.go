// Package tlsdialer contains code to establish TLS connections.
package tlsdialer

import (
	"context"
	"crypto/tls"
	"net"
)

// UnderlyingDialer is the underlying dialer type.
type UnderlyingDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// TLSHandshaker is the generic TLS handshaker
type TLSHandshaker interface {
	Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}
