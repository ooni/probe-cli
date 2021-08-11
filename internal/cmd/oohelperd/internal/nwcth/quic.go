package nwcth

import (
	"context"
	"crypto/tls"

	"github.com/apex/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

// QUICDo performs the QUIC check.
func QUICDo(ctx context.Context, endpoint string, tlsConf *tls.Config) (quic.EarlySession, *TLSHandshakeMeasurement) {
	quicdialer := netx.NewQUICDialer(netx.Config{Logger: log.Log})
	sess, err := quicdialer.DialContext(ctx, "udp", endpoint, tlsConf, &quic.Config{})
	if err != nil {
		s := err.Error()
		return nil, &TLSHandshakeMeasurement{
			Failure: &s,
		}
	}
	return sess, &TLSHandshakeMeasurement{}
}
