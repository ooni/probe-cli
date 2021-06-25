package netxmocks

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
)

// QUICListener is a mockable netxlite.QUICListener.
type QUICListener struct {
	MockListen func(addr *net.UDPAddr) (net.PacketConn, error)
}

// Listen calls MockListen.
func (ql *QUICListener) Listen(addr *net.UDPAddr) (net.PacketConn, error) {
	return ql.MockListen(addr)
}

// QUICContextDialer is a mockable netxlite.QUICContextDialer.
type QUICContextDialer struct {
	MockDialContext func(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)
}

// DialContext calls MockDialContext.
func (qcd *QUICContextDialer) DialContext(ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	return qcd.MockDialContext(ctx, network, address, tlsConfig, quicConfig)
}
