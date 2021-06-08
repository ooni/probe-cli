package dialer

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

// ErrorWrapperDialer is a dialer that performs err wrapping
type ErrorWrapperDialer struct {
	Dialer
}

// DialContext implements Dialer.DialContext
func (d ErrorWrapperDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	err = errorx.SafeErrWrapperBuilder{
		Error:     err,
		Operation: errorx.ConnectOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
	}
	return &ErrorWrapperConn{Conn: conn}, nil
}

// ErrorWrapperConn is a net.Conn that performs error wrapping.
type ErrorWrapperConn struct {
	net.Conn
}

// Read implements net.Conn.Read
func (c ErrorWrapperConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	err = errorx.SafeErrWrapperBuilder{
		Error:     err,
		Operation: errorx.ReadOperation,
	}.MaybeBuild()
	return
}

// Write implements net.Conn.Write
func (c ErrorWrapperConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	err = errorx.SafeErrWrapperBuilder{
		Error:     err,
		Operation: errorx.WriteOperation,
	}.MaybeBuild()
	return
}

// Close implements net.Conn.Close
func (c ErrorWrapperConn) Close() (err error) {
	err = c.Conn.Close()
	err = errorx.SafeErrWrapperBuilder{
		Error:     err,
		Operation: errorx.CloseOperation,
	}.MaybeBuild()
	return
}
