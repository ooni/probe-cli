package quicdialer

import (
	"context"
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
)

// ContextDialer is a dialer for QUIC using Context.
type ContextDialer interface {
	// Note: assumes that tlsCfg and cfg are not nil.
	DialContext(ctx context.Context, network, host string,
		tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error)
}

// Dialer dials QUIC connections.
type Dialer interface {
	// Note: assumes that tlsCfg and cfg are not nil.
	Dial(network, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error)
}

// Resolver is the interface we expect from a resolver.
type Resolver interface {
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
}
