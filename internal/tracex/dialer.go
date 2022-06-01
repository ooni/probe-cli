package tracex

//
// TCP and connected UDP sockets
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DialerSaver saves events occurring during the dial
type DialerSaver struct {
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
	return &dialerConnectObserver{
		saver: s,
	}
}

type dialerConnectObserver struct {
	saver *Saver
}

var _ model.DialerWrapper = &dialerConnectObserver{}

func (w *dialerConnectObserver) WrapDialer(d model.Dialer) model.Dialer {
	return &DialerSaver{
		Dialer: d,
		Saver:  w.saver,
	}
}

// DialContext implements Dialer.DialContext
func (d *DialerSaver) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	stop := time.Now()
	d.Saver.Write(&EventConnectOperation{&EventValue{
		Address:  address,
		Duration: stop.Sub(start),
		Err:      err,
		Proto:    network,
		Time:     stop,
	}})
	return conn, err
}

func (d *DialerSaver) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// DialerConnSaver wraps the returned connection such that we
// collect all the read/write events that occur.
type DialerConnSaver struct {
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
	return &dialerReadWriteObserver{
		saver: s,
	}
}

type dialerReadWriteObserver struct {
	saver *Saver
}

var _ model.DialerWrapper = &dialerReadWriteObserver{}

func (w *dialerReadWriteObserver) WrapDialer(d model.Dialer) model.Dialer {
	return &DialerConnSaver{
		Dialer: d,
		Saver:  w.saver,
	}
}

// DialContext implements Dialer.DialContext
func (d *DialerConnSaver) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &dialerConnWrapper{saver: d.Saver, Conn: conn}, nil
}

func (d *DialerConnSaver) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

type dialerConnWrapper struct {
	net.Conn
	saver *Saver
}

func (c *dialerConnWrapper) Read(p []byte) (int, error) {
	proto := c.Conn.RemoteAddr().Network()
	remoteAddr := c.Conn.RemoteAddr().String()
	start := time.Now()
	count, err := c.Conn.Read(p)
	stop := time.Now()
	c.saver.Write(&EventReadOperation{&EventValue{
		Address:  remoteAddr,
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Proto:    proto,
		Time:     stop,
	}})
	return count, err
}

func (c *dialerConnWrapper) Write(p []byte) (int, error) {
	proto := c.Conn.RemoteAddr().Network()
	remoteAddr := c.Conn.RemoteAddr().String()
	start := time.Now()
	count, err := c.Conn.Write(p)
	stop := time.Now()
	c.saver.Write(&EventWriteOperation{&EventValue{
		Address:  remoteAddr,
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Proto:    proto,
		Time:     stop,
	}})
	return count, err
}

var _ model.Dialer = &DialerSaver{}
var _ model.Dialer = &DialerConnSaver{}
var _ net.Conn = &dialerConnWrapper{}
