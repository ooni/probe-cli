package netxlite

import (
	"context"
	"crypto/x509"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// MaybeCustomUnderlyingNetwork is a nil-safe [model.UnderlyingNetwork] provider. When the pointer
// to the [MaybeCustomUnderlyingNetwork] is nil or the underlying field is nil, the Get method of the
// [MaybeCustomUnderlyingNetwork] falls back to calling [tproxySingleton].
type MaybeCustomUnderlyingNetwork struct {
	underlying model.UnderlyingNetwork
}

// Get returns the [model.UnderlyingNetwork] returned by [tproxySingleton] if p is nil or the
// underlying field is nil and otherwise returns the value of the underlying field.
func (p *MaybeCustomUnderlyingNetwork) Get() model.UnderlyingNetwork {
	if p == nil || p.underlying == nil {
		return tproxySingleton()
	}
	return p.underlying
}

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

// tproxyDefaultCertPool is a static copy of the default cert pool. You
// MUST NOT access this variable directly. You SHOULD use the
// tproxySingleton().DefaultCertPool() factory instead. By doing
// that, you would allow integration tests to override the pool
// we're using. Hence, we can run tests with fake servers.
//
// See https://github.com/ooni/probe/issues/2413 to understand why we
// need a private static default pool.
var tproxyDefaultCertPool = NewMozillaCertPool()

// DefaultCertPool implements model.UnderlyingNetwork
func (tp *DefaultTProxy) DefaultCertPool() *x509.CertPool {
	return tproxyDefaultCertPool
}

const defaultDialTimeout = 15 * time.Second

// DialTimeout implements model.UnderlyingNetwork
func (tp *DefaultTProxy) DialTimeout() time.Duration {
	return defaultDialTimeout
}

// DialContext implements UnderlyingNetwork.
func (tp *DefaultTProxy) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d := &net.Dialer{}
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
