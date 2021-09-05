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
	Resolver   netxlite.ResolverLegacy
	TLSConf    *tls.Config
}

// QUICDo performs the QUIC check.
func QUICDo(ctx context.Context, config QUICConfig) (quic.EarlySession, error) {
	if config.QUICDialer != nil {
		return config.QUICDialer.DialContext(ctx, "udp", config.Endpoint, config.TLSConf, &quic.Config{})
	}
	resolver := config.Resolver
	if resolver == nil {
		resolver = &netxlite.ResolverSystem{}
	}
	dialer := NewQUICDialerResolver(resolver)
	return dialer.DialContext(ctx, "udp", config.Endpoint, config.TLSConf, &quic.Config{})
}
