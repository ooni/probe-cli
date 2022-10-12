package netxlite

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// TProxy refers to the UnderlyingNetwork implementation. By overriding this
// variable you can force netxlite to use alternative network primitives.
var TProxy model.UnderlyingNetwork = &DefaultTProxy{}

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
