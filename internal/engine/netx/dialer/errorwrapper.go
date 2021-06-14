package dialer

import (
	"context"
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
		return nil, NewErrDial(&err)
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
		return n, NewErrRead(&err)
	}
	return
}

// Write implements net.Conn.Write
func (c ErrorWrapperConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if err != nil {
		return n, NewErrWrite(&err)
	}
	return
}

// Close implements net.Conn.Close
func (c ErrorWrapperConn) Close() (err error) {
	err = c.Conn.Close()
	if err != nil {
		return NewErrClose(&err)
	}
	return
}

type ErrDial struct {
	error
}

func NewErrDial(e *error) *ErrDial {
	return &ErrDial{*e}
}

func (e *ErrDial) Unwrap() error {
	return e.error
}

type ErrWrite struct {
	error
}

func NewErrWrite(e *error) *ErrWrite {
	return &ErrWrite{*e}
}

func (e *ErrWrite) Unwrap() error {
	return e.error
}

type ErrRead struct {
	error
}

func NewErrRead(e *error) *ErrRead {
	return &ErrRead{*e}
}

func (e *ErrRead) Unwrap() error {
	return e.error
}

type ErrClose struct {
	error
}

func NewErrClose(e *error) *ErrClose {
	return &ErrClose{*e}
}

func (e *ErrClose) Unwrap() error {
	return e.error
}
