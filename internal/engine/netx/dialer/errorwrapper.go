package dialer

import (
	"context"
	"errors"
	"net"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/dialid"
)

// ErrorWrapperDialer is a dialer that performs err wrapping
type ErrorWrapperDialer struct {
	Dialer
}

// DialContext implements Dialer.DialContext
func (d ErrorWrapperDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialID := dialid.ContextDialID(ctx)
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, &ErrDial{err}
	}
	return &ErrorWrapperConn{
		Conn: conn, ConnID: safeConnID(network, conn), DialID: dialID}, nil
}

// ErrorWrapperConn is a net.Conn that performs error wrapping.
type ErrorWrapperConn struct {
	net.Conn
	ConnID int64
	DialID int64
}

// Read implements net.Conn.Read
func (c ErrorWrapperConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if err != nil {
		return n, &ErrRead{err}
	}
	return
}

// Write implements net.Conn.Write
func (c ErrorWrapperConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if err != nil {
		return n, &ErrWrite{err}
	}
	return
}

// Close implements net.Conn.Close
func (c ErrorWrapperConn) Close() (err error) {
	err = c.Conn.Close()
	if err != nil {
		return &ErrClose{err}
	}
	return
}

// TODO(kelmenhorst): why do we use different types here? maybe just one struct with a field indicating the operation? this would avoid using errors.As..
type ErrDial struct {
	error
}

func (e *ErrDial) Unwrap() error {
	return e.error
}

func NewErrDial(e error) *ErrDial {
	return &ErrDial{e}
}

type ErrWrite struct {
	error
}

func (e *ErrWrite) Unwrap() error {
	return e.error
}

type ErrRead struct {
	error
}

func (e *ErrRead) Unwrap() error {
	return e.error
}

type ErrClose struct {
	error
}

func (e *ErrClose) Unwrap() error {
	return e.error
}

// export for for testing purposes
var MockErrDial ErrDial = ErrDial{errors.New("mock error")}
var MockErrRead ErrRead = ErrRead{errors.New("mock error")}
var MockErrWrite ErrWrite = ErrWrite{errors.New("mock error")}
var MockErrClose ErrClose = ErrClose{errors.New("mock error")}
