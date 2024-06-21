package netxlite

//
// Code to ensure we have proper read timeouts (for reliability
// as described by https://github.com/ooni/probe/issues/1609)
//

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// httpDialerWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpDialerWithReadTimeout struct {
	Dialer model.Dialer
}

var _ model.Dialer = &httpDialerWithReadTimeout{}

func (d *httpDialerWithReadTimeout) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// DialContext implements Dialer.DialContext.
func (d *httpDialerWithReadTimeout) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &httpConnWithReadTimeout{conn}, nil
}

// httpTLSDialerWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpTLSDialerWithReadTimeout struct {
	TLSDialer model.TLSDialer
}

var _ model.TLSDialer = &httpTLSDialerWithReadTimeout{}

func (d *httpTLSDialerWithReadTimeout) CloseIdleConnections() {
	d.TLSDialer.CloseIdleConnections()
}

// ErrNotTLSConn occur when an interface accepts a net.Conn but
// internally needs a TLSConn and you pass a net.Conn that doesn't
// implement TLSConn to such an interface.
var ErrNotTLSConn = errors.New("not a TLSConn")

// DialTLSContext implements TLSDialer's DialTLSContext.
func (d *httpTLSDialerWithReadTimeout) DialTLSContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.TLSDialer.DialTLSContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	tconn, okay := conn.(TLSConn) // part of the contract but let's be graceful
	if !okay {
		_ = conn.Close() // we own the conn here
		return nil, ErrNotTLSConn
	}
	return &httpTLSConnWithReadTimeout{tconn}, nil
}

// httpConnWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpConnWithReadTimeout struct {
	net.Conn
}

// httpConnReadTimeout is the read timeout we apply to all HTTP
// conns (see https://github.com/ooni/probe/issues/1609).
//
// This timeout is meant as a fallback mechanism so that a stuck
// connection will _eventually_ fail. This is why it is set to
// a large value (300 seconds when writing this note).
//
// There should be other mechanisms to ensure that the code is
// lively: the context during the RoundTrip and iox.ReadAllContext
// when reading the body. They should kick in earlier. But we
// additionally want to avoid leaking a (parked?) connection and
// the corresponding goroutine, hence this large timeout.
//
// A future @bassosimone may understand this problem even better
// and possibly apply an even better fix to this issue. This
// will happen when we'll be able to further study the anomalies
// described in https://github.com/ooni/probe/issues/1609.
const httpConnReadTimeout = 300 * time.Second

// Read implements Conn.Read.
func (c *httpConnWithReadTimeout) Read(b []byte) (int, error) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(httpConnReadTimeout))
	defer c.Conn.SetReadDeadline(time.Time{})
	return c.Conn.Read(b)
}

// httpTLSConnWithReadTimeout enforces a read timeout for all HTTP
// connections. See https://github.com/ooni/probe/issues/1609.
type httpTLSConnWithReadTimeout struct {
	TLSConn
}

// Read implements Conn.Read.
func (c *httpTLSConnWithReadTimeout) Read(b []byte) (int, error) {
	_ = c.TLSConn.SetReadDeadline(time.Now().Add(httpConnReadTimeout))
	defer c.TLSConn.SetReadDeadline(time.Time{})
	return c.TLSConn.Read(b)
}
