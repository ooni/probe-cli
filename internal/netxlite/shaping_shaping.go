//go:build shaping

package netxlite

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func newMaybeShapingDialer(dialer model.Dialer) model.Dialer {
	return &shapingDialer{dialer}
}

type shapingDialer struct {
	model.Dialer
}

// DialContext implements Dialer.DialContext
func (d *shapingDialer) DialContext(
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

func (c *shapingConn) Read(p []byte) (int, error) {
	time.Sleep(100 * time.Millisecond)
	return c.Conn.Read(p)
}

func (c *shapingConn) Write(p []byte) (int, error) {
	time.Sleep(100 * time.Millisecond)
	return c.Conn.Write(p)
}
