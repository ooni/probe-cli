package nwcth

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func newDialer() netxlite.Dialer {
	// TODO(bassosimone,kelmenhorst): what complexity do we need here for the dialer? is this enough?
	var d netxlite.Dialer = netxlite.DefaultDialer
	d = &errorsx.ErrorWrapperDialer{Dialer: d}
	return d
}

// TCPDo performs the TCP check.
func TCPDo(ctx context.Context, endpoint string, dialer netxlite.Dialer) (net.Conn, error) {
	// TODO(bassosimone,kelmenhorst): do we need the complexity of a netx dialer here? is net.Dial enough?
	return dialer.DialContext(ctx, "tcp", endpoint)
}
