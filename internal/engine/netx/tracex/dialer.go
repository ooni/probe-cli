package tracex

//
// TCP and connected UDP sockets
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// SaverDialer saves events occurring during the dial
type SaverDialer struct {
	// Dialer is the underlying dialer,
	Dialer model.Dialer

	// Saver saves events.
	Saver *Saver
}

// NewConnectObserver returns a DialerWrapper that observes the
// connect event. This function will return nil, which is a valid
// DialerWrapper for netxlite.WrapDialer, if Saver is nil.
func (s *Saver) NewConnectObserver() model.DialerWrapper {
	if s == nil {
		return nil // valid DialerWrapper according to netxlite's docs
	}
	return &saverDialerWrapper{
		saver: s,
	}
}

type saverDialerWrapper struct {
	saver *Saver
}

var _ model.DialerWrapper = &saverDialerWrapper{}

func (w *saverDialerWrapper) WrapDialer(d model.Dialer) model.Dialer {
	return &SaverDialer{
		Dialer: d,
		Saver:  w.saver,
	}
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

func (d *SaverDialer) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// SaverConnDialer wraps the returned connection such that we
// collect all the read/write events that occur.
type SaverConnDialer struct {
	// Dialer is the underlying dialer
	Dialer model.Dialer

	// Saver saves events
	Saver *Saver
}

// NewReadWriteObserver returns a DialerWrapper that observes the
// I/O events. This function will return nil, which is a valid
// DialerWrapper for netxlite.WrapDialer, if Saver is nil.
func (s *Saver) NewReadWriteObserver() model.DialerWrapper {
	if s == nil {
		return nil // valid DialerWrapper according to netxlite's docs
	}
	return &saverReadWriteWrapper{
		saver: s,
	}
}

type saverReadWriteWrapper struct {
	saver *Saver
}

var _ model.DialerWrapper = &saverReadWriteWrapper{}

func (w *saverReadWriteWrapper) WrapDialer(d model.Dialer) model.Dialer {
	return &SaverConnDialer{
		Dialer: d,
		Saver:  w.saver,
	}
}

// DialContext implements Dialer.DialContext
func (d *SaverConnDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &saverConn{saver: d.Saver, Conn: conn}, nil
}

func (d *SaverConnDialer) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
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
