package websteps

import (
	"context"
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

type QUICConfig struct {
	Endpoint   string
	QUICDialer netxlite.QUICContextDialer
	Resolver   netxlite.Resolver
	TLSConf    *tls.Config
}

// QUICDo performs the QUIC check.
func QUICDo(ctx context.Context, config QUICConfig) (quic.EarlySession, error) {
	if config.QUICDialer != nil {
		return config.QUICDialer.DialContext(ctx, "udp", config.Endpoint, config.TLSConf, &quic.Config{})
	}
	dialer := NewQUICDialerResolver(config.Resolver)
	return dialer.DialContext(ctx, "udp", config.Endpoint, config.TLSConf, &quic.Config{})
}
