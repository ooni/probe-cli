// go:build windows

package tlsmiddlebox

//
// syscall utilities for dialerTTLWrapperConn
//

import (
	"net"
	"strings"
	"syscall"
)

// SetTTL sets the IP TTL field for the underlying net.TCPConn
func (c *dialerTTLWrapperConn) SetTTL(ttl int) error {
	conn := c.Conn
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return errInvalidConnWrapper
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}
	rawErr := rawConn.Control(func(fd uintptr) {
		isIPv6 := strings.Contains(tcpConn.RemoteAddr().String(), "[")
		if isIPv6 {
			err = syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, ttl)
		} else {
			err = syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
		}
	})
	// The syscall err is given a higher priority and returned early if non-nil
	if err != nil {
		return err
	}
	return rawErr
}

// GetSoErr fetches the SO_ERROR value at look for soft ICMP errors in TCP
func (c *dialerTTLWrapperConn) GetSoErr() (int, error) {
	var cErrno int
	conn := c.Conn
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return 0, errInvalidConnWrapper
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return 0, errInvalidConnWrapper
	}
	rawErr := rawConn.Control(func(fd uintptr) {
		cErrno = getErrFromSockOpt(fd)
	})
	if rawErr != nil {
		return 0, rawErr
	}
	return cErrno, nil
}
