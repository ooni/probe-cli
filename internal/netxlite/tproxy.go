package netxlite

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// TProxable is the fundamental type used by the netxlite package to perform
// low-level network operations for which, by default, we use the stdlib.
//
// The t stands for transparent. By using this type as the fundamental type,
// we can transparently intercept connections and implement censorship
// policies. The implementation of this functionality is not part of netxlite:
// here we only have the basic mechanism to make this possible.
type TProxable interface {
	// Name is the name of the implementation.
	Name() string

	// ListenUDP creates a new quicx.UDPLikeConn conn.
	ListenUDP(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error)

	// LookupHost lookups a domain using the stdlib resolver.
	LookupHost(ctx context.Context, domain string) ([]string, error)

	// NewTProxyDialer returns a new TProxyDialer.
	NewTProxyDialer(timeout time.Duration) TProxyDialer
}

// TProxyDialer is the dialer type returned by TProxable.NewDialer.
type TProxyDialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// TProxy is the fundamental variable controlling how netxlite creates
// net.Conn and quicx.UDPLikeConn, as well as how it uses the stdlib
// resolver. By modifying this variable, you can effectively transparently
// proxy netxlite (and hence OONI) activities to other services. This is
// quite convenient when performing quality assurance.
var TProxy TProxable = &TProxyStdlib{}

// TProxyStdlib is the default TProxable implementation that uses
// the stdlib in the most obvious way for every functionality.
type TProxyStdlib struct{}

// Name returns the name of this tproxy.
func (*TProxyStdlib) Name() string {
	return "stdlib"
}

// ListenUDP calls net.ListenUDP.
func (*TProxyStdlib) ListenUDP(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	return net.ListenUDP(network, laddr)
}

// LookupHost calls net.DefaultResolver.LookupHost.
func (*TProxyStdlib) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, domain)
}

// NewTProxyDialer returns a &net.Dialer{Timeout: timeout} instance.
func (*TProxyStdlib) NewTProxyDialer(timeout time.Duration) TProxyDialer {
	return &net.Dialer{Timeout: timeout}
}
