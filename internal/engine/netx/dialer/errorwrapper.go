package dialer

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/dialid"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

// ErrorWrapperDialer is a dialer that performs err wrapping
type ErrorWrapperDialer struct {
	Dialer
}

// DialContext implements Dialer.DialContext
func (d ErrorWrapperDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialID := dialid.ContextDialID(ctx)
	conn, err := d.Dialer.DialContext(ctx, network, address)
	err = errorx.SafeErrWrapperBuilder{
		// ConnID does not make any sense if we've failed and the error
		// does not make any sense (and is nil) if we succeded.
		DialID:    dialID,
		Error:     err,
		Operation: errorx.ConnectOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
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
	err = errorx.SafeErrWrapperBuilder{
		ConnID:    c.ConnID,
		DialID:    c.DialID,
		Error:     err,
		Operation: errorx.ReadOperation,
	}.MaybeBuild()
	return
}

// Write implements net.Conn.Write
func (c ErrorWrapperConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	err = errorx.SafeErrWrapperBuilder{
		ConnID:    c.ConnID,
		DialID:    c.DialID,
		Error:     err,
		Operation: errorx.WriteOperation,
	}.MaybeBuild()
	return
}

// Close implements net.Conn.Close
func (c ErrorWrapperConn) Close() (err error) {
	err = c.Conn.Close()
	err = errorx.SafeErrWrapperBuilder{
		ConnID:    c.ConnID,
		DialID:    c.DialID,
		Error:     err,
		Operation: errorx.CloseOperation,
	}.MaybeBuild()
	return
}
