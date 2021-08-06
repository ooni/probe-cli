package nwcth

import (
	"context"
	"crypto/tls"

	"github.com/apex/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

// QUICConfig configures the QUIC handshake check.
type QUICConfig struct {
	Dialer    netx.QUICDialer
	Endpoint  string
	QConfig   *quic.Config
	TLSConfig *tls.Config
}

// QUICDo performs the QUIC check.
func QUICDo(ctx context.Context, config *QUICConfig) (quic.EarlySession, *TLSHandshakeMeasurement) {
	quicdialer := netx.NewQUICDialer(netx.Config{Logger: log.Log})
	sess, err := quicdialer.DialContext(ctx, "udp", config.Endpoint, config.TLSConfig, config.QConfig)
	return sess, &TLSHandshakeMeasurement{
		Failure: newfailure(err),
	}
}
