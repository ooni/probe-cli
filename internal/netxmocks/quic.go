package netxmocks

import "net"

// QUICListener is a mockable netxlite.QUICListener.
type QUICListener struct {
	MockListen func(addr *net.UDPAddr) (net.PacketConn, error)
}

// Listen calls MockListen.
func (ql *QUICListener) Listen(addr *net.UDPAddr) (net.PacketConn, error) {
	return ql.MockListen(addr)
}
