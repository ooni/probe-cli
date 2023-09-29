package pnet

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Connect returns a [Stage] that creates TCP/UDP connections.
func Connect() Stage[Endpoint, NetConn] {
	return stageForAction[Endpoint, NetConn](actionFunc[Endpoint, NetConn](connectAction))
}

// connectAction is the [action] that connects to a given endpoint.
func connectAction(ctx context.Context, endpoint Endpoint, outputs chan<- Result[NetConn]) {
	// create the destination address
	addrport := net.JoinHostPort(endpoint.IPAddress, endpoint.Port)

	// start the operation logger
	ol := logx.NewOperationLogger(endpoint.Logger, "Connect %s/%s", addrport, endpoint.Network)

	// create dialer
	dialer := netxlite.NewDialerWithoutResolver(endpoint.Logger)

	// connect
	conn, err := dialer.DialContext(ctx, endpoint.Network, addrport)

	// stop the operation logger
	ol.Stop(err)

	// handle the error case
	if err != nil {
		outputs <- NewResultError[NetConn](err)
		return
	}

	// handle the successful case
	res := NetConn{Conn: conn, Logger: endpoint.Logger}
	outputs <- NewResultValue(res)
}
