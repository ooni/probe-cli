package dialer

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/connid"
)

// Dialer is the interface we expect from a dialer
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Resolver is the interface we expect from a resolver
type Resolver interface {
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
}

func safeLocalAddress(conn net.Conn) (s string) {
	if conn != nil && conn.LocalAddr() != nil {
		s = conn.LocalAddr().String()
	}
	return
}

func safeConnID(network string, conn net.Conn) int64 {
	return connid.Compute(network, safeLocalAddress(conn))
}
