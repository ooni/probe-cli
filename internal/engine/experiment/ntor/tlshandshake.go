package ntor

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/ooni/probe-cli/v3/internal/measuring/tlshandshaker"
)

// doTLSHandshake performs a TLS handshake with "or_port" or "or_port_dirauth".
func (svc *service) doTLSHandshake(ctx context.Context, out *serviceOutput, conn net.Conn) {
	defer conn.Close() // we own it
	tlsConn, err := svc.tlsHandshaker.Handshake(ctx, &tlshandshaker.HandshakeRequest{
		Conn: conn,
		Config: &tls.Config{
			InsecureSkipVerify: true,
		},
		Logger: svc.logger,
		Saver:  &out.saver,
	})
	if err != nil {
		out.err = err
		out.operation = "tls_handshake"
		return
	}
	tlsConn.Close()
}
