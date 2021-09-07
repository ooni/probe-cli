package errorsx

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// Dialer establishes network connections.
type Dialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// ErrorWrapperDialer is a dialer that performs error wrapping. The connection
// returned by the DialContext function will also perform error wrapping.
type ErrorWrapperDialer struct {
	// Dialer is the underlying dialer.
	Dialer
}

// DialContext implements Dialer.DialContext.
func (d *ErrorWrapperDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, SafeErrWrapperBuilder{
			Classifier: errorsx.ClassifyGenericError,
			Operation:  errorsx.ConnectOperation,
			Error:      err,
		}.MaybeBuild()
	}
	return &errorWrapperConn{Conn: conn}, nil
}

// errorWrapperConn is a net.Conn that performs error wrapping.
type errorWrapperConn struct {
	// Conn is the underlying connection.
	net.Conn
}

// Read implements net.Conn.Read.
func (c *errorWrapperConn) Read(b []byte) (int, error) {
	count, err := c.Conn.Read(b)
	if err != nil {
		return 0, SafeErrWrapperBuilder{
			Classifier: errorsx.ClassifyGenericError,
			Operation:  errorsx.ReadOperation,
			Error:      err,
		}.MaybeBuild()
	}
	return count, nil
}

// Write implements net.Conn.Write.
func (c *errorWrapperConn) Write(b []byte) (int, error) {
	count, err := c.Conn.Write(b)
	if err != nil {
		return 0, SafeErrWrapperBuilder{
			Classifier: errorsx.ClassifyGenericError,
			Operation:  errorsx.WriteOperation,
			Error:      err,
		}.MaybeBuild()
	}
	return count, nil
}

// Close implements net.Conn.Close.
func (c *errorWrapperConn) Close() error {
	err := c.Conn.Close()
	if err != nil {
		return SafeErrWrapperBuilder{
			Classifier: errorsx.ClassifyGenericError,
			Operation:  errorsx.CloseOperation,
			Error:      err,
		}.MaybeBuild()
	}
	return nil
}
