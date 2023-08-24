package netxlite

import (
	"context"
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// NetemUnderlyingNetworkAdapter adapts [netem.UnderlyingNetwork] to [model.UnderlyingNetwork].
type NetemUnderlyingNetworkAdapter struct {
	UNet netem.UnderlyingNetwork
}

var _ model.UnderlyingNetwork = &NetemUnderlyingNetworkAdapter{}

// DefaultCertPool implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) DefaultCertPool() *x509.CertPool {
	return runtimex.Try1(a.UNet.DefaultCertPool())
}

// DialContext implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) DialContext(ctx context.Context, timeout time.Duration, network string, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return a.UNet.DialContext(ctx, network, address)
}

// GetaddrinfoLookupANY implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) GetaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	return a.UNet.GetaddrinfoLookupANY(ctx, domain)
}

// GetaddrinfoResolverNetwork implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) GetaddrinfoResolverNetwork() string {
	return a.UNet.GetaddrinfoResolverNetwork()
}

// ListenUDP implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) ListenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return a.UNet.ListenUDP(network, addr)
}
