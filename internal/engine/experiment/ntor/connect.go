package ntor

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/measuring/connector"
)

// doConnect establishes a TCP connection to the given endpoint. We will
// perform this action for any available target type.
func (svc *service) doConnect(ctx context.Context, out *serviceOutput) {
	conn, err := svc.connector.DialContext(ctx, &connector.DialRequest{
		Network: "tcp",
		Address: out.results.TargetAddress,
		Logger:  svc.logger,
		Saver:   &out.saver,
	})
	if err != nil {
		out.err = err
		out.operation = "connect"
		return
	}
	switch out.results.TargetProtocol {
	case "dir_port":
		svc.doHTTPGet(ctx, out, conn)
	case "or_port", "or_port_dirauth":
		svc.doTLSHandshake(ctx, out, conn)
	case "obfs4":
		svc.doOBFS4(ctx, out, conn)
	default:
		conn.Close() // we own the connection
	}
}
