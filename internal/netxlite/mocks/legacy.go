package mocks

import (
	"context"
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
)

// QUICContextDialer is a mockable netxlite.QUICContextDialer.
//
// DEPRECATED: please use QUICDialer.
type QUICContextDialer struct {
	MockDialContext func(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error)
}

// DialContext calls MockDialContext.
func (qcd *QUICContextDialer) DialContext(ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
	return qcd.MockDialContext(ctx, network, address, tlsConfig, quicConfig)
}
