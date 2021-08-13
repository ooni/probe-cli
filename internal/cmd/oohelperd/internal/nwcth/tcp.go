package nwcth

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// newDialer contructs a new dialer for TCP connections,
// with default, errorwrapping and resolve functionalities
func newDialerResolver(resolver netxlite.Resolver) netxlite.Dialer {
	var d netxlite.Dialer = netxlite.DefaultDialer
	d = &errorsx.ErrorWrapperDialer{Dialer: d}
	d = &netxlite.DialerResolver{Resolver: resolver, Dialer: d}
	return d
}

// TCPDo performs the TCP check.
func TCPDo(ctx context.Context, endpoint string, dialer netxlite.Dialer) (net.Conn, error) {
	return dialer.DialContext(ctx, "tcp", endpoint)
}
