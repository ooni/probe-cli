package dslnet

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/dslmodel"
	"github.com/ooni/probe-cli/v3/internal/logx"
)

// Connect establishes a TCP or UDP connection.
func Connect(ctx context.Context, rt dslmodel.Runtime, endpoint Endpoint) (NetConn, error) {
	// start the operation logger
	addrport := net.JoinHostPort(endpoint.IPAddress, endpoint.Port)
	traceID := rt.NewTraceID()
	ol := logx.NewOperationLogger(rt.Logger(), "trace#%d: Connect %s/%s", traceID, addrport, endpoint.Network)

	// create trace for collecting OONI observations
	trace := rt.NewTrace(traceID, rt.ZeroTime(), endpoint.Tags...)

	// establish the connection
	dialer := trace.NewDialerWithoutResolver(rt.Logger())
	conn, err := dialer.DialContext(ctx, endpoint.Network, addrport)

	// stop the operation logger
	ol.Stop(err)

	return NetConn{conn}, err
}

// ConnectPipeline returns a [dslmodel.Pipeline] that calls [Connect].
func ConnectPipeline() dslmodel.Pipeline[Endpoint, NetConn] {
	return dslmodel.FilterToPipeline(dslmodel.SyncOperationToFilter(
		dslmodel.FunctionWithScalarResultToSyncOperation(Connect),
	))
}
