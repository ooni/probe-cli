package measurex

//
// Dialer
//
// Wrappers for Dialer and Conn to store events into a WritableDB.
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Conn is a network connection.
type Conn = net.Conn

// Dialer dials network connections.
type Dialer = netxlite.Dialer

// WrapDialer creates a new dialer that writes events
// into the given WritableDB. The net.Conns created by
// a wrapped dialer also write into the WritableDB.
func (mx *Measurer) WrapDialer(db WritableDB, dialer netxlite.Dialer) Dialer {
	return WrapDialer(mx.Begin, db, dialer)
}

// WrapDialer wraps a dialer.
func WrapDialer(begin time.Time, db WritableDB, dialer netxlite.Dialer) Dialer {
	return &dialerDB{Dialer: dialer, db: db, begin: begin}
}

// NewDialerWithSystemResolver creates a
func (mx *Measurer) NewDialerWithSystemResolver(db WritableDB, logger Logger) Dialer {
	r := mx.NewResolverSystem(db, logger)
	return mx.WrapDialer(db, netxlite.NewDialerWithResolver(logger, r))
}

// NewDialerWithoutResolver is a convenience factory for creating
// a dialer that saves measurements into the DB and that is not attached
// to any resolver (hence only works when passed IP addresses).
func (mx *Measurer) NewDialerWithoutResolver(db WritableDB, logger Logger) Dialer {
	return mx.WrapDialer(db, netxlite.NewDialerWithoutResolver(logger))
}

type dialerDB struct {
	netxlite.Dialer
	begin time.Time
	db    WritableDB
}

// NetworkEvent contains a network event. This kind of events
// are generated by Dialer, QUICDialer, Conn, QUICConn.
type NetworkEvent struct {
	// JSON names compatible with df-008-netevents
	RemoteAddr string  `json:"address"`
	Failure    *string `json:"failure"`
	Count      int     `json:"num_bytes,omitempty"`
	Operation  string  `json:"operation"`
	Network    string  `json:"proto"`
	Finished   float64 `json:"t"`
	Started    float64 `json:"started"`

	// Names that are not part of the spec.
	Oddity Oddity `json:"oddity"`
}

func (d *dialerDB) DialContext(
	ctx context.Context, network, address string) (Conn, error) {
	started := time.Since(d.begin).Seconds()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	finished := time.Since(d.begin).Seconds()
	d.db.InsertIntoDial(&NetworkEvent{
		Operation:  "connect",
		Network:    network,
		RemoteAddr: address,
		Started:    started,
		Finished:   finished,
		Failure:    NewArchivalFailure(err),
		Oddity:     d.computeOddity(err),
		Count:      0,
	})
	if err != nil {
		return nil, err
	}
	return &connDB{
		Conn:       conn,
		begin:      d.begin,
		db:         d.db,
		network:    network,
		remoteAddr: address,
	}, nil
}

func (c *dialerDB) computeOddity(err error) Oddity {
	if err == nil {
		return ""
	}
	switch err.Error() {
	case netxlite.FailureGenericTimeoutError:
		return OddityTCPConnectTimeout
	case netxlite.FailureConnectionRefused:
		return OddityTCPConnectRefused
	case netxlite.FailureHostUnreachable:
		return OddityTCPConnectHostUnreachable
	default:
		return OddityTCPConnectOher
	}
}

type connDB struct {
	net.Conn
	begin      time.Time
	db         WritableDB
	network    string
	remoteAddr string
}

func (c *connDB) Read(b []byte) (int, error) {
	started := time.Since(c.begin).Seconds()
	count, err := c.Conn.Read(b)
	finished := time.Since(c.begin).Seconds()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Operation:  "read",
		Network:    c.network,
		RemoteAddr: c.remoteAddr,
		Started:    started,
		Finished:   finished,
		Failure:    NewArchivalFailure(err),
		Count:      count,
	})
	return count, err
}

func (c *connDB) Write(b []byte) (int, error) {
	started := time.Since(c.begin).Seconds()
	count, err := c.Conn.Write(b)
	finished := time.Since(c.begin).Seconds()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Operation:  "write",
		Network:    c.network,
		RemoteAddr: c.remoteAddr,
		Started:    started,
		Finished:   finished,
		Failure:    NewArchivalFailure(err),
		Count:      count,
	})
	return count, err
}

func (c *connDB) Close() error {
	started := time.Since(c.begin).Seconds()
	err := c.Conn.Close()
	finished := time.Since(c.begin).Seconds()
	c.db.InsertIntoClose(&NetworkEvent{
		Operation:  "close",
		Network:    c.network,
		RemoteAddr: c.remoteAddr,
		Started:    started,
		Finished:   finished,
		Failure:    NewArchivalFailure(err),
		Count:      0,
	})
	return err
}
