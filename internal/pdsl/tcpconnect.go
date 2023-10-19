package pdsl

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
)

// TCPConn is the [net.Conn] produced by [TCPConnect].
type TCPConn struct {
	Trace Trace
	net.Conn
}

// TCPConnect returns a [Filter] that attempts to create [TCPConn] from [Endpoint].
func TCPConnect(ctx context.Context, rt Runtime, tags ...string) Filter[Endpoint, TCPConn] {
	return func(mendpoints <-chan Result[Endpoint]) <-chan Result[TCPConn] {
		outputs := make(chan Result[TCPConn])

		go func() {
			// make sure we close the outputs channel
			defer close(outputs)

			for mendpoint := range mendpoints {
				// handle the case of previous stage failure
				if err := mendpoint.Err; err != nil {
					outputs <- NewResultError[TCPConn](err)
					continue
				}
				endpoint := mendpoint.Value

				// start the operation logger
				traceID := rt.NewTraceID()
				ol := logx.NewOperationLogger(rt.Logger(), "[#%d] TCPConnect %s", traceID, endpoint)

				// create trace for collecting OONI observations
				trace := rt.NewTrace(traceID, rt.ZeroTime(), tags...)

				// enforce a timeout
				const timeout = 15 * time.Second
				ctx, cancel := context.WithTimeout(ctx, timeout)

				// establish the connection
				dialer := trace.NewDialerWithoutResolver(rt.Logger())
				conn, err := dialer.DialContext(ctx, "tcp", string(endpoint))

				// cancel the context
				cancel()

				// stop the operation logger
				ol.Stop(err)

				// handle failure
				if err != nil {
					outputs <- NewResultError[TCPConn](err)
					continue
				}

				// make sure the Runtime eventually closes the connection
				rt.RegisterCloser(conn)

				// handle success
				outputs <- NewResultValue(TCPConn{trace, conn})
			}
		}()

		return outputs
	}
}
