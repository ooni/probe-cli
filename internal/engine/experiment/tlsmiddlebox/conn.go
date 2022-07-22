package tlsmiddlebox

import (
	"errors"
	"net"
	"syscall"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// dialerTTLWrapperConn wraps errors as well as allows us to set the TTL
type dialerTTLWrapperConn struct {
	net.Conn
}

var _ net.Conn = &dialerTTLWrapperConn{}

func (c *dialerTTLWrapperConn) Read(b []byte) (int, error) {
	count, err := c.Conn.Read(b)
	if err != nil {
		return 0, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
	}
	return count, nil
}

func (c *dialerTTLWrapperConn) Write(b []byte) (int, error) {
	count, err := c.Conn.Write(b)
	if err != nil {
		return 0, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
	}
	return count, nil
}

func (c *dialerTTLWrapperConn) Close() error {
	err := c.Conn.Close()
	if err != nil {
		return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.CloseOperation, err)
	}
	return nil
}

// setTTL calls setConnTTL to set the TTL for a net.TCPConn
// Note: The passed conn must be of type dialerTTLWrapperConn
func setTTL(conn net.Conn, ttl int) error {
	ttlWrapper, ok := conn.(*dialerTTLWrapperConn)
	if !ok {
		return errors.New("invalid TTL wrapper for conn")
	}
	return setConnTTL(ttlWrapper.Conn, ttl)
}

// setConnTTL sets the IP TTL field for a net.TCPConn
// Note: The passed conn must be of type net.TCPConn
func setConnTTL(conn net.Conn, ttl int) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return errors.New("underlying conn is not of type net.TCPConn")
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}
	err = rawConn.Control(func(fd uintptr) {
		setTTLSyscall(int(fd), ttl)
	})
	return err
}

// setTTLSyscall is the syscall to set the TTL of a file descriptor
func setTTLSyscall(fd int, ttl int) error {
	return syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
}
