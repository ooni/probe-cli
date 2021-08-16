package websteps

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

type TCPConfig struct {
	dialer   netxlite.Dialer
	endpoint string
	resolver netxlite.Resolver
}

// TCPDo performs the TCP check.
func TCPDo(ctx context.Context, config TCPConfig) (net.Conn, error) {
	if config.dialer != nil {
		return config.dialer.DialContext(ctx, "tcp", config.endpoint)
	}
	resolver := config.resolver
	if resolver == nil {
		resolver = &netxlite.ResolverSystem{}
	}
	dialer := NewDialerResolver(resolver)
	return dialer.DialContext(ctx, "tcp", config.endpoint)
}
