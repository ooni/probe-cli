package dialer

import (
	"context"
	"net"
)

// errorWrapperDialer is a dialer that performs err wrapping
type errorWrapperDialer struct {
	Dialer
}

// DialContext implements Dialer.DialContext
func (d *errorWrapperDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, NewErrDial(&err)
	}
	return &errorWrapperConn{Conn: conn}, nil
}

// errorWrapperConn is a net.Conn that performs error wrapping.
type errorWrapperConn struct {
	net.Conn
}

// Read implements net.Conn.Read
func (c *errorWrapperConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if err != nil {
		return n, NewErrRead(&err)
	}
	return
}

// Write implements net.Conn.Write
func (c *errorWrapperConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if err != nil {
		return n, NewErrWrite(&err)
	}
	return
}

// Close implements net.Conn.Close
func (c *errorWrapperConn) Close() (err error) {
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
