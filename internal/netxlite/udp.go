package netxlite

import (
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewUDPListener creates a new UDPListener using the underlying
// [*Netx] structure to create listening UDP sockets.
func (netx *Netx) NewUDPListener() model.UDPListener {
	return &udpListenerErrWrapper{&udpListenerStdlib{provider: netx.MaybeCustomUnderlyingNetwork()}}
}

// udpListenerStdlib is a UDPListener using the standard library.
type udpListenerStdlib struct {
	// provider is the OPTIONAL nil-safe [model.UnderlyingNetwork] provider.
	provider *MaybeCustomUnderlyingNetwork
}

var _ model.UDPListener = &udpListenerStdlib{}

// Listen implements UDPListener.Listen.
func (qls *udpListenerStdlib) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return qls.provider.Get().ListenUDP("udp", addr)
}
