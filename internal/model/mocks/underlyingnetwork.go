package mocks

import (
	"context"
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// UnderlyingNetwork allows mocking model.UnderlyingNetwork.
type UnderlyingNetwork struct {
	MockDialContext func(ctx context.Context, timeout time.Duration, network, address string) (net.Conn, error)

	MockListenUDP func(network string, addr *net.UDPAddr) (model.UDPLikeConn, error)

	MockGetaddrinfoLookupANY func(ctx context.Context, domain string) ([]string, string, error)

	MockGetaddrinfoResolverNetwork func() string

	MockMaybeModifyPool func(pool *x509.CertPool) *x509.CertPool
}

var _ model.UnderlyingNetwork = &UnderlyingNetwork{}

func (un *UnderlyingNetwork) DialContext(ctx context.Context, timeout time.Duration, network, address string) (net.Conn, error) {
	return un.MockDialContext(ctx, timeout, network, address)
}

func (un *UnderlyingNetwork) ListenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return un.MockListenUDP(network, addr)
}

func (un *UnderlyingNetwork) GetaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	return un.MockGetaddrinfoLookupANY(ctx, domain)
}

func (un *UnderlyingNetwork) GetaddrinfoResolverNetwork() string {
	return un.MockGetaddrinfoResolverNetwork()
}

// MaybeModifyPool implements model.UnderlyingNetwork
func (un *UnderlyingNetwork) MaybeModifyPool(pool *x509.CertPool) *x509.CertPool {
	return un.MockMaybeModifyPool(pool)
}
