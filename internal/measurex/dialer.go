package measurex

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// Conn is the connection type we use.
type Conn interface {
	net.Conn

	// ConnID returns the connection ID.
	ConnID() int64
}

// Dialer is the dialer type we use.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (Conn, error)
	CloseIdleConnections()
}

// WrapDialer wraps a Dialer to add measurex capabilities.
//
// DialContext algorithm
//
// 1. perform TCP/UDP connect as usual;
//
// 2. insert a DialEvent into the DB;
//
// 3. on success, wrap the returned net.Conn so that it
// inserts Read, Write, and Close events into the DB.
//
// 4. return net.Conn or error.
func WrapDialer(origin Origin, db DB, d netxlite.Dialer) Dialer {
	return &dialerx{Dialer: d, db: db, origin: origin}

}

type dialerx struct {
	netxlite.Dialer
	db     DB
	origin Origin
}

// NetworkEvent contains a network event.
type NetworkEvent struct {
	Origin        Origin
	MeasurementID int64
	ConnID        int64
	Operation     string
	Network       string
	RemoteAddr    string
	LocalAddr     string
	Started       time.Time
	Finished      time.Time
	Error         error
	Oddity        Oddity
	Count         int
}

func (d *dialerx) DialContext(
	ctx context.Context, network, address string) (Conn, error) {
	connID := d.db.NextConnID()
	started := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	finished := time.Now()
	d.db.InsertIntoDial(&NetworkEvent{
		Origin:        d.origin,
		MeasurementID: d.db.MeasurementID(),
		ConnID:        connID,
		Operation:     "connect",
		Network:       network,
		RemoteAddr:    address,
		LocalAddr:     d.localAddrIfNotNil(conn),
		Started:       started,
		Finished:      finished,
		Error:         err,
		Oddity:        d.computeOddity(err),
		Count:         0,
	})
	if err != nil {
		return nil, err
	}
	return &connx{
		Conn:       conn,
		db:         d.db,
		connID:     connID,
		remoteAddr: address,
		localAddr:  conn.LocalAddr().String(),
		network:    network,
		origin:     d.origin,
	}, nil
}

func (c *dialerx) localAddrIfNotNil(conn net.Conn) (addr string) {
	if conn != nil {
		addr = conn.LocalAddr().String()
	}
	return
}

func (c *dialerx) computeOddity(err error) Oddity {
	if err == nil {
		return ""
	}
	switch err.Error() {
	case errorsx.FailureGenericTimeoutError:
		return OddityTCPConnectTimeout
	case errorsx.FailureConnectionRefused:
		return OddityTCPConnectRefused
	default:
		return OddityTCPConnectOher
	}
}

type connx struct {
	net.Conn
	db         DB
	connID     int64
	remoteAddr string
	localAddr  string
	network    string
	origin     Origin
}

func (c *connx) ConnID() int64 {
	return c.connID
}

func (c *connx) Read(b []byte) (int, error) {
	started := time.Now()
	count, err := c.Conn.Read(b)
	finished := time.Now()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Origin:        c.origin,
		MeasurementID: c.db.MeasurementID(),
		ConnID:        c.connID,
		Operation:     "read",
		Network:       c.network,
		RemoteAddr:    c.remoteAddr,
		LocalAddr:     c.localAddr,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Count:         count,
	})
	return count, err
}

func (c *connx) Write(b []byte) (int, error) {
	started := time.Now()
	count, err := c.Conn.Write(b)
	finished := time.Now()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Origin:        c.origin,
		MeasurementID: c.db.MeasurementID(),
		ConnID:        c.connID,
		Operation:     "write",
		Network:       c.network,
		RemoteAddr:    c.remoteAddr,
		LocalAddr:     c.localAddr,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Count:         count,
	})
	return count, err
}

func (c *connx) Close() error {
	started := time.Now()
	err := c.Conn.Close()
	finished := time.Now()
	c.db.InsertIntoClose(&NetworkEvent{
		Origin:        c.origin,
		MeasurementID: c.db.MeasurementID(),
		ConnID:        c.connID,
		Operation:     "close",
		Network:       c.network,
		RemoteAddr:    c.remoteAddr,
		LocalAddr:     c.localAddr,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Count:         0,
	})
	return err
}
