package bytecounter

import "net"

// Conn wraps a network connection and counts bytes.
type Conn struct {
	// net.Conn is the underlying net.Conn.
	net.Conn

	// Counter is the byte counter.
	Counter *Counter
}

// Read implements net.Conn.Read.
func (c *Conn) Read(p []byte) (int, error) {
	count, err := c.Conn.Read(p)
	c.Counter.CountBytesReceived(count)
	return count, err
}

// Write implements net.Conn.Write.
func (c *Conn) Write(p []byte) (int, error) {
	count, err := c.Conn.Write(p)
	c.Counter.CountBytesSent(count)
	return count, err
}
