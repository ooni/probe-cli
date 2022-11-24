package netxlite

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// TProxySet sets the value of the transparent proxy in a thread safe way
// and returns the previous value just in case you want to restore it later.
//
// CAVEAT: it's not recommended to modify the transparent proxy while OONI's
// doing network I/O. Please, only use TProxySet for integration testing.
func TProxySet(t model.UnderlyingNetwork) model.UnderlyingNetwork {
	tproxyMu.Lock()
	oldt := tproxy
	tproxy = t
	tproxyMu.Unlock()
	return oldt
}

var (
	// tproxy refers to the UnderlyingNetwork implementation used by netxlite.
	tproxy model.UnderlyingNetwork = &DefaultTProxy{}

	// tproxyMu protects tproxy
	tproxyMu sync.Mutex
)

// defaultTProxy is the default UnderlyingNetwork implementation.
type DefaultTProxy struct{}

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
