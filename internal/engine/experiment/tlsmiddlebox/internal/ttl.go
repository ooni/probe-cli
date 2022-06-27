package internal

import (
	"errors"
	"net"
	"syscall"
)

// ResetConnTTL resets the TTL to its default value of 64 or a
// sufficiently high value to prevent ICMP Time Exceeded
func ResetConnTTL(conn net.Conn) (err error) {
	err = SetConnTTL(conn, 64)
	return
}

// SetConnTTL sets the conn TTL to the required value
func SetConnTTL(conn net.Conn, ttl int) error {
	TCPConn, ok := conn.(*net.TCPConn)
	if !ok {
		return errors.New("type cast failed")
	}
	RawConn, err := TCPConn.SyscallConn()
	if err != nil {
		return err
	}
	err = RawConn.Control(func(fd uintptr) {
		setTTL(int(fd), ttl)
	})
	return err
}

// SetTTL is the syscall to set the IP TTL field of the file desccriptor obtained from net.conn
func setTTL(fd int, ttl int) error {
	return syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
}
