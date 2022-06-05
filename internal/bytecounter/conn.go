package bytecounter

//
// Code to wrap a net.Conn
//

import "net"

// wrappedConn wraps a network connection and counts bytes.
type wrappedConn struct {
	// net.Conn is the underlying net.Conn.
	net.Conn

	// Counter is the byte counter.
	Counter *Counter
}

// Read implements net.Conn.Read.
func (c *wrappedConn) Read(p []byte) (int, error) {
	count, err := c.Conn.Read(p)
	c.Counter.CountBytesReceived(count)
	return count, err
}

// Write implements net.Conn.Write.
func (c *wrappedConn) Write(p []byte) (int, error) {
	count, err := c.Conn.Write(p)
	c.Counter.CountBytesSent(count)
	return count, err
}

// WrapConn returns a new conn that uses the given counter.
func WrapConn(conn net.Conn, counter *Counter) net.Conn {
	return &wrappedConn{Conn: conn, Counter: counter}
}

// MaybeWrapConn is like wrap if counter is not nil, otherwise it's a no-op.
func MaybeWrapConn(conn net.Conn, counter *Counter) net.Conn {
	if counter == nil {
		return conn
	}
	return WrapConn(conn, counter)
}
