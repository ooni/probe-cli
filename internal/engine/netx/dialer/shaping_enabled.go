// +build shaping

package dialer

import (
	"context"
	"net"
	"time"
)

// ShapingDialer ensures we don't use too much bandwidth
// when using integration tests at GitHub. To select
// the implementation with shaping use `-tags shaping`.
type ShapingDialer struct {
	Dialer
}

// DialContext implements Dialer.DialContext
func (d ShapingDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &shapingConn{Conn: conn}, nil
}

type shapingConn struct {
	net.Conn
}

func (c shapingConn) Read(p []byte) (int, error) {
	time.Sleep(100 * time.Millisecond)
	return c.Conn.Read(p)
}

func (c shapingConn) Write(p []byte) (int, error) {
	time.Sleep(100 * time.Millisecond)
	return c.Conn.Write(p)
}
