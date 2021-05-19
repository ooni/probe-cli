package netplumbing

import (
	"net"
)

// quicUDPConnWrapper wraps an udpConn connection used by QUIC.
type quicUDPConnWrapper struct {
	byteCounter ByteCounter
	*net.UDPConn
}

// ErrReadFrom is a readFrom error.
type ErrReadFrom struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrReadFrom) Unwrap() error {
	return err.error
}

// ReadMsgUDP reads a message from an UDP socket.
func (conn *quicUDPConnWrapper) ReadMsgUDP(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
	n, oobn, flags, addr, err := conn.UDPConn.ReadMsgUDP(b, oob)
	if err != nil {
		return 0, 0, 0, nil, &ErrReadFrom{err}
	}
	conn.byteCounter.CountBytesReceived(n + oobn)
	return n, oobn, flags, addr, nil
}

// ErrWriteTo is a writeTo error.
type ErrWriteTo struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrWriteTo) Unwrap() error {
	return err.error
}

// WriteTo writes a message to the UDP socket.
func (conn *quicUDPConnWrapper) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := conn.UDPConn.WriteTo(p, addr)
	if err != nil {
		return 0, &ErrWriteTo{err}
	}
	conn.byteCounter.CountBytesSent(count)
	return count, nil
}
