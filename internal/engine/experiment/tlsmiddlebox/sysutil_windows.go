// go:build cgo && windows

package tlsmiddlebox

//
// syscall utilities for dialerTTLWrapperConn
//

// #cgo LDFLAGS: -lws2_32
// #include <winsock2.h>
import "C"

import (
	"net"
	"syscall"
	"unsafe"
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
	err = rawConn.Control(func(fd uintptr) {
		syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	})
	return err
}

// GetSoErr fetches the SO_ERROR value at look for soft ICMP errors in TCP
func (c *dialerTTLWrapperConn) GetSoErr() (int, error) {
	var cErrno C.int
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
		szInt := C.sizeof_int
		C.getsockopt((C.SOCKET)(fd), (C.int)(C.SOL_SOCKET), (C.int)(C.SO_ERROR), (*C.char)(unsafe.Pointer(&cErrno)), (*C.int)(unsafe.Pointer(&szInt)))
	})
	if rawErr != nil {
		return 0, rawErr
	}
	return int(cErrno), nil
}
