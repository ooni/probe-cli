package tlsmiddlebox

//
// Wrapped TTL conn
//

import (
	"errors"
	"net"
	"syscall"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

var errInvalidConnWrapper = errors.New("invalid conn wrapper")

// setConnTTL calls SetTTL to set the TTL for a dialerTTLWrapperConn
func setConnTTL(conn net.Conn, ttl int) error {
	ttlWrapper, ok := conn.(*dialerTTLWrapperConn)
	if !ok {
		return errInvalidConnWrapper
	}
	return ttlWrapper.SetTTL(ttl)
}

// getSoErr calls GetSoErr to fetch the SO_ERROR value
func getSoErr(conn net.Conn) (soErr error, err error) {
	ttlWrapper, ok := conn.(*dialerTTLWrapperConn)
	if !ok {
		return nil, errInvalidConnWrapper
	}
	errno, err := ttlWrapper.GetSoErr()
	if err != nil {
		return nil, err
	}
	return syscall.Errno(errno), nil
}

// dialerTTLWrapperConn wraps errors as well as allows us to set the TTL
type dialerTTLWrapperConn struct {
	net.Conn
}

var _ net.Conn = &dialerTTLWrapperConn{}

// Read implements net.Conn.Read
func (c *dialerTTLWrapperConn) Read(b []byte) (int, error) {
	count, err := c.Conn.Read(b)
	if err != nil {
		return 0, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
	}
	return count, nil
}

// Write implements net.Conn.Write
func (c *dialerTTLWrapperConn) Write(b []byte) (int, error) {
	count, err := c.Conn.Write(b)
	if err != nil {
		return 0, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
	}
	return count, nil
}

// Close implements net.Conn.Close
func (c *dialerTTLWrapperConn) Close() error {
	err := c.Conn.Close()
	if err != nil {
		return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.CloseOperation, err)
	}
	return nil
}
