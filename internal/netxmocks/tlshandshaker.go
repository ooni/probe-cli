package netxmocks

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
