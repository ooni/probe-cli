//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package tlsmiddlebox

import (
	"net"
	"syscall"
)

// SetTTL sets the IP TTL field for the underlying net.TCPConn
func (c *dialerTTLWrapperConn) SetTTL(ttl int) error {
	conn := c.Conn
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return ErrInvalidConnWrapper
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

// GetSoErr fetches the SO_ERROR value to look for soft ICMP errors in TCP
func (c *dialerTTLWrapperConn) GetSoErr() (errno int, err error) {
	conn := c.Conn
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return 0, ErrInvalidConnWrapper
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return 0, ErrInvalidConnWrapper
	}
	rawErr := rawConn.Control(func(fd uintptr) {
		errno, err = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_ERROR)
	})
	if rawErr != nil {
		return 0, rawErr
	}
	return
}
