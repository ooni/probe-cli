package tlsmiddlebox

import (
	"errors"
	"net"
	"syscall"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// setConnTTL calls SetTTL to set the TTL for a dialerTTLWrapperConn
func setConnTTL(conn net.Conn, ttl int) error {
	ttlWrapper, ok := conn.(*dialerTTLWrapperConn)
	if !ok {
		return errors.New("invalid TTL wrapper for conn")
	}
	return ttlWrapper.SetTTL(ttl)
}

// getSoErr fetches the SO_ERROR for a dialerTTLWrapperConn
func getSoErr(conn net.Conn) error {
	ttlWrapper, ok := conn.(*dialerTTLWrapperConn)
	if !ok {
		return errors.New("invalid TTL wrapper for conn")
	}
	errno, err := ttlWrapper.GetSoError()
	if err == nil {
		err = syscall.Errno(errno)
	}
	return err
}

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

// SetTTL sets the IP TTL field for the underlying net.TCPConn
func (c *dialerTTLWrapperConn) SetTTL(ttl int) error {
	conn := c.Conn
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return errors.New("underlying conn is not of type net.TCPConn")
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}
	err = rawConn.Control(func(fd uintptr) {
		syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	})
	return err
}

// GetSoError gets the SO_ERROR for the underlying net.TCPConn
func (c *dialerTTLWrapperConn) GetSoError() (errno int, err error) {
	conn := c.Conn
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return -1, errors.New("underlying conn is not of type net.TCPConn")
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return -1, err
	}
	rawConn.Control(func(fd uintptr) {
		errno, err = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_ERROR)
	})
	return
}
