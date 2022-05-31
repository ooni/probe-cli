package tracex

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// SaverDialer saves events occurring during the dial
type SaverDialer struct {
	model.Dialer
	Saver *Saver
}

// DialContext implements Dialer.DialContext
func (d *SaverDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	stop := time.Now()
	d.Saver.Write(Event{
		Address:  address,
		Duration: stop.Sub(start),
		Err:      err,
		Name:     netxlite.ConnectOperation,
		Proto:    network,
		Time:     stop,
	})
	return conn, err
}

// SaverConnDialer wraps the returned connection such that we
// collect all the read/write events that occur.
type SaverConnDialer struct {
	model.Dialer
	Saver *Saver
}

// DialContext implements Dialer.DialContext
func (d *SaverConnDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &saverConn{saver: d.Saver, Conn: conn}, nil
}

type saverConn struct {
	net.Conn
	saver *Saver
}

func (c *saverConn) Read(p []byte) (int, error) {
	start := time.Now()
	count, err := c.Conn.Read(p)
	stop := time.Now()
	c.saver.Write(Event{
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     netxlite.ReadOperation,
		Time:     stop,
	})
	return count, err
}

func (c *saverConn) Write(p []byte) (int, error) {
	start := time.Now()
	count, err := c.Conn.Write(p)
	stop := time.Now()
	c.saver.Write(Event{
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     netxlite.WriteOperation,
		Time:     stop,
	})
	return count, err
}

var _ model.Dialer = &SaverDialer{}
var _ model.Dialer = &SaverConnDialer{}
var _ net.Conn = &saverConn{}
