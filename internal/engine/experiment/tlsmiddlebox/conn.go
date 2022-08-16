package tlsmiddlebox

import (
	"errors"
	"net"

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
