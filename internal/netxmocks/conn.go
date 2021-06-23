package netxmocks

import (
	"net"
	"time"
)

// Conn is a mockable net.Conn.
type Conn struct {
	MockRead             func(b []byte) (int, error)
	MockWrite            func(b []byte) (int, error)
	MockClose            func() error
	MockLocalAddr        func() net.Addr
	MockRemoteAddr       func() net.Addr
	MockSetDeadline      func(t time.Time) error
	MockSetReadDeadline  func(t time.Time) error
	MockSetWriteDeadline func(t time.Time) error
}

// Read implements net.Conn.Read
func (c *Conn) Read(b []byte) (int, error) {
	return c.MockRead(b)
}

// Write implements net.Conn.Write
func (c *Conn) Write(b []byte) (int, error) {
	return c.MockWrite(b)
}

// Close implements net.Conn.Close
func (c *Conn) Close() error {
	return c.MockClose()
}

// LocalAddr returns the local address
func (c *Conn) LocalAddr() net.Addr {
	return c.MockLocalAddr()
}

// RemoteAddr returns the remote address
func (c *Conn) RemoteAddr() net.Addr {
	return c.MockRemoteAddr()
}

// SetDeadline sets the connection deadline.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.MockSetDeadline(t)
}

// SetReadDeadline sets the read deadline.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.MockSetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.MockSetWriteDeadline(t)
}

var _ net.Conn = &Conn{}
