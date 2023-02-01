package netxlite

import (
	"context"
	"crypto/x509"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// tproxySingletonInst refers to the UnderlyingNetwork implementation. By overriding this
// variable you can force netxlite to use alternative network primitives.
var tproxySingletonInst model.UnderlyingNetwork = &DefaultTProxy{}

// tproxyMu protects the tproxySingleton.
var tproxyMu sync.Mutex

// WithCustomTProxy runs the given function with a different UnderlyingNetwork
// and restores the previous UnderlyingNetwork before returning.
func WithCustomTProxy(tproxy model.UnderlyingNetwork, function func()) {
	tproxyMu.Lock()
	orig := tproxySingletonInst
	tproxySingletonInst = tproxy
	tproxyMu.Unlock()
	defer func() {
		tproxyMu.Lock()
		tproxySingletonInst = orig
		tproxyMu.Unlock()
	}()
	function()
}

// tproxySingleton returns the tproxySingleton in a goroutine-safe way.
func tproxySingleton() model.UnderlyingNetwork {
	defer tproxyMu.Unlock()
	tproxyMu.Lock()
	return tproxySingletonInst
}

// DefaultTProxy is the default UnderlyingNetwork implementation.
type DefaultTProxy struct{}

// MaybeModifyPool implements model.UnderlyingNetwork
func (tp *DefaultTProxy) MaybeModifyPool(pool *x509.CertPool) *x509.CertPool {
	return pool
}

// DialContext implements UnderlyingNetwork.
func (tp *DefaultTProxy) DialContext(ctx context.Context, timeout time.Duration, network, address string) (net.Conn, error) {
	d := &net.Dialer{
		Timeout: timeout,
	}
	return d.DialContext(ctx, network, address)
}

// ListenUDP implements UnderlyingNetwork.
func (tp *DefaultTProxy) ListenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return net.ListenUDP(network, addr)
}

// GetaddrinfoLookupANY implements UnderlyingNetwork.
func (tp *DefaultTProxy) GetaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	return getaddrinfoLookupANY(ctx, domain)
}

// GetaddrinfoResolverNetwork implements UnderlyingNetwork.
func (tp *DefaultTProxy) GetaddrinfoResolverNetwork() string {
	return getaddrinfoResolverNetwork()
}
