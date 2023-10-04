package netxlite

import (
	"context"
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NetemUnderlyingNetworkAdapter adapts [netem.UnderlyingNetwork] to [model.UnderlyingNetwork].
type NetemUnderlyingNetworkAdapter struct {
	UNet netem.UnderlyingNetwork
}

var _ model.UnderlyingNetwork = &NetemUnderlyingNetworkAdapter{}

// DefaultCertPool implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) DefaultCertPool() *x509.CertPool {
	return a.UNet.DefaultCertPool()
}

// DialTimeout implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) DialTimeout() time.Duration {
	return defaultDialTimeout
}

// DialContext implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
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

// ListenTCP implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) ListenTCP(network string, addr *net.TCPAddr) (net.Listener, error) {
	return a.UNet.ListenTCP(network, addr)
}

// ListenUDP implements model.UnderlyingNetwork
func (a *NetemUnderlyingNetworkAdapter) ListenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return a.UNet.ListenUDP(network, addr)
}
