package netemx

//
// Code to adapt [netem.UnderlyingNetwork] to [model.UnderlyingNetwork].
//

import (
	"context"
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// WithCustomTProxy executes the given function using the given [netem.UnderlyingNetwork]
// as the [model.UnderlyingNetwork] used by the [netxlite] package.
func WithCustomTProxy(tproxy netem.UnderlyingNetwork, function func()) {
	netxlite.WithCustomTProxy(&adapter{tproxy}, function)
}

// adapter adapts [netem.UnderlyingNetwork] to [model.UnderlyingNetwork].
type adapter struct {
	tp netem.UnderlyingNetwork
}

var _ model.UnderlyingNetwork = &adapter{}

// DefaultCertPool implements model.UnderlyingNetwork
func (a *adapter) DefaultCertPool() *x509.CertPool {
	return runtimex.Try1(a.tp.DefaultCertPool())
}

// DialContext implements model.UnderlyingNetwork
func (a *adapter) DialContext(ctx context.Context, timeout time.Duration, network string, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return a.tp.DialContext(ctx, network, address)
}

// GetaddrinfoLookupANY implements model.UnderlyingNetwork
func (a *adapter) GetaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	return a.tp.GetaddrinfoLookupANY(ctx, domain)
}

// GetaddrinfoResolverNetwork implements model.UnderlyingNetwork
func (a *adapter) GetaddrinfoResolverNetwork() string {
	return a.tp.GetaddrinfoResolverNetwork()
}

// ListenUDP implements model.UnderlyingNetwork
func (a *adapter) ListenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return a.tp.ListenUDP(network, addr)
}
