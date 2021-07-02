package errorsx

import (
	"context"
	"net"
)

// Dialer establishes network connections.
type Dialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// ErrorWrapperDialer is a dialer that performs err wrapping.
type ErrorWrapperDialer struct {
	Dialer
}

// DialContext implements Dialer.DialContext
func (d *ErrorWrapperDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	err = SafeErrWrapperBuilder{
		Error:     err,
		Operation: ConnectOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
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
	err = SafeErrWrapperBuilder{
		Error:     err,
		Operation: ReadOperation,
	}.MaybeBuild()
	return
}

// Write implements net.Conn.Write
func (c *errorWrapperConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	err = SafeErrWrapperBuilder{
		Error:     err,
		Operation: WriteOperation,
	}.MaybeBuild()
	return
}

// Close implements net.Conn.Close
func (c *errorWrapperConn) Close() (err error) {
	err = c.Conn.Close()
	err = SafeErrWrapperBuilder{
		Error:     err,
		Operation: CloseOperation,
	}.MaybeBuild()
	return
}
