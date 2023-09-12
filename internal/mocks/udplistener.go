package mocks

import (
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// UDPListener is a mockable netxlite.UDPListener.
type UDPListener struct {
	MockListen func(addr *net.UDPAddr) (model.UDPLikeConn, error)
}

// Listen calls MockListen.
func (ql *UDPListener) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return ql.MockListen(addr)
}
