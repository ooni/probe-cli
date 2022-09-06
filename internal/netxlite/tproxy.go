package netxlite

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// TProxyDialWithDialer is the top-level function used for dialing. By default we use
// the given dialer, but you can override it. Should you choose to override this function,
// please ensure you're honouring dialer.Timeout, if nonzero, when dialing.
var TProxyDialWithDialer = func(ctx context.Context, d *net.Dialer, network, address string) (net.Conn, error) {
	return d.DialContext(ctx, network, address)
}

// TProxyListenUDP is the top-level function used to create listening UDP connections. By default
// this function calls net.ListenUDP, but you can override it.
var TProxyListenUDP = func(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return net.ListenUDP(network, addr)
}

// TProxyGetaddrinfoLookupANY is the toplevel function used to invoke getaddrinfo. By default
// this function calls getaddrinfoLookupANY, but you can override it.
var TProxyGetaddrinfoLookupANY = func(ctx context.Context, domain string) ([]string, string, error) {
	return getaddrinfoLookupANY(ctx, domain)
}
