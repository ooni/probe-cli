package dialer

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

// saverDialer saves events occurring during the dial
type saverDialer struct {
	Dialer
	Saver *trace.Saver
}

// DialContext implements Dialer.DialContext
func (d *saverDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	stop := time.Now()
	d.Saver.Write(trace.Event{
		Address:  address,
		Duration: stop.Sub(start),
		Err:      err,
		Name:     errorsx.ConnectOperation,
		Proto:    network,
		Time:     stop,
	})
	return conn, err
}

// saverConnDialer wraps the returned connection such that we
// collect all the read/write events that occur.
type saverConnDialer struct {
	Dialer
	Saver *trace.Saver
}

// DialContext implements Dialer.DialContext
func (d *saverConnDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &saverConn{saver: d.Saver, Conn: conn}, nil
}

type saverConn struct {
	net.Conn
	saver *trace.Saver
}

func (c *saverConn) Read(p []byte) (int, error) {
	start := time.Now()
	count, err := c.Conn.Read(p)
	stop := time.Now()
	c.saver.Write(trace.Event{
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     errorsx.ReadOperation,
		Time:     stop,
	})
	return count, err
}

func (c *saverConn) Write(p []byte) (int, error) {
	start := time.Now()
	count, err := c.Conn.Write(p)
	stop := time.Now()
	c.saver.Write(trace.Event{
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     errorsx.WriteOperation,
		Time:     stop,
	})
	return count, err
}

var _ Dialer = &saverDialer{}
var _ Dialer = &saverConnDialer{}
var _ net.Conn = &saverConn{}
