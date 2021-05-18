package netplumbing

import "net"

// connWrapper is a wrapper for net.Conn.
type connWrapper struct {
	byteCounter ByteCounter
	net.Conn
}

// ErrRead is a read error.
type ErrRead struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrRead) Unwrap() error {
	return err.error
}

// Read implements net.Conn.Read. When this function returns an
// error it's always an ErrRead error.
func (conn *connWrapper) Read(b []byte) (int, error) {
	count, err := conn.Conn.Read(b)
	if err != nil {
		return 0, &ErrRead{err}
	}
	conn.byteCounter.CountBytesReceived(count)
	return count, nil
}

// ErrWrite is a write error.
type ErrWrite struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrWrite) Unwrap() error {
	return err.error
}

// Write implements net.Conn.Write. When this function returns an
// error, it's always an ErrWrite error.
func (conn *connWrapper) Write(b []byte) (int, error) {
	count, err := conn.Conn.Write(b)
	if err != nil {
		return 0, &ErrWrite{err}
	}
	conn.byteCounter.CountBytesSent(count)
	return count, nil
}
