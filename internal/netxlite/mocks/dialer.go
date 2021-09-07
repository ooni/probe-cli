package mocks

import (
	"context"
	"net"
	"time"
)

// Dialer is a mockable Dialer.
type Dialer struct {
	MockDialContext          func(ctx context.Context, network, address string) (net.Conn, error)
	MockCloseIdleConnections func()
}

// DialContext calls MockDialContext.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.MockDialContext(ctx, network, address)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (d *Dialer) CloseIdleConnections() {
	d.MockCloseIdleConnections()
}

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

// Read calls MockRead.
func (c *Conn) Read(b []byte) (int, error) {
	return c.MockRead(b)
}

// Write calls MockWrite.
func (c *Conn) Write(b []byte) (int, error) {
	return c.MockWrite(b)
}

// Close calls MockClose.
func (c *Conn) Close() error {
	return c.MockClose()
}

// LocalAddr calls MockLocalAddr.
func (c *Conn) LocalAddr() net.Addr {
	return c.MockLocalAddr()
}

// RemoteAddr calls MockRemoteAddr.
func (c *Conn) RemoteAddr() net.Addr {
	return c.MockRemoteAddr()
}

// SetDeadline calls MockSetDeadline.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.MockSetDeadline(t)
}

// SetReadDeadline calls MockSetReadDeadline.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.MockSetReadDeadline(t)
}

// SetWriteDeadline calls MockSetWriteDeadline.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.MockSetWriteDeadline(t)
}

var _ net.Conn = &Conn{}
