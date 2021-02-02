package resolver

import (
	"context"
	"crypto/tls"
	"net"
)

func DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	connch := make(chan net.Conn)
	errch := make(chan error, 1)
	go func() {
		conn, err := tls.Dial(network, address, new(tls.Config))
		if err != nil {
			errch <- err
			return
		}
		select {
		case <-ctx.Done():
			conn.Close()
		case connch <- conn:
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn := <-connch:
		return conn, nil
	case err := <-errch:
		return nil, err
	}
}
