package netxlite

//
// Transparent proxy (for integration testing)
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// TProxy is the fundamental variable controlling how netxlite creates
// net.Conn and model.UDPLikeConn, as well as how it uses the stdlib
// resolver. By modifying this variable, you can effectively transparently
// proxy netxlite (and hence OONI) activities to other services. This is
// quite convenient when performing quality assurance tests.
var TProxy model.UnderlyingNetworkLibrary = &TProxyStdlib{}

// TProxyStdlib is the default model.UnderlyingNetworkLibrary using
// the stdlib in the most obvious way for every functionality.
type TProxyStdlib struct{}

// ListenUDP calls net.ListenUDP.
func (*TProxyStdlib) ListenUDP(network string, laddr *net.UDPAddr) (model.UDPLikeConn, error) {
	return net.ListenUDP(network, laddr)
}

// LookupHost calls net.DefaultResolver.LookupHost.
func (*TProxyStdlib) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, domain)
}

// NewSimpleDialer returns a &net.Dialer{Timeout: timeout} instance.
func (*TProxyStdlib) NewSimpleDialer(timeout time.Duration) model.SimpleDialer {
	return &net.Dialer{Timeout: timeout}
}
