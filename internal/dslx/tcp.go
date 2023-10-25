package dslx

//
// TCP measurements
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
)

// TCPConnect returns a function that establishes TCP connections.
func TCPConnect(rt Runtime) Func[*Endpoint, *TCPConnection] {
	return Operation[*Endpoint, *TCPConnection](func(ctx context.Context, input *Endpoint) *Maybe[*TCPConnection] {
		// create trace
		trace := rt.NewTrace(rt.IDGenerator().Add(1), rt.ZeroTime(), input.Tags...)

		// start the operation logger
		ol := logx.NewOperationLogger(
			rt.Logger(),
			"[#%d] TCPConnect %s",
			trace.Index(),
			input.Address,
		)

		// setup
		const timeout = 15 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// obtain the dialer to use
		dialer := trace.NewDialerWithoutResolver(rt.Logger())

		// connect
		conn, err := dialer.DialContext(ctx, "tcp", input.Address)

		// possibly register established conn for late close
		rt.MaybeTrackConn(conn)

		// stop the operation logger
		ol.Stop(err)

		// save the observations
		rt.SaveObservations(maybeTraceToObservations(trace)...)

		state := &TCPConnection{
			Address: input.Address,
			Conn:    conn, // possibly nil
			Domain:  input.Domain,
			Network: input.Network,
			Trace:   trace,
		}

		return &Maybe[*TCPConnection]{
			Error:        err,
			Observations: maybeTraceToObservations(trace),
			State:        state,
		}
	})
}

// TCPConnection is an established TCP connection. If you initialize
// manually, init at least the ones marked as MANDATORY.
type TCPConnection struct {
	// Address is the MANDATORY address we tried to connect to.
	Address string

	// Conn is the established connection.
	Conn net.Conn

	// Domain is the OPTIONAL domain from which we resolved the Address.
	Domain string

	// Network is the MANDATORY network we tried to use when connecting.
	Network string

	// Trace is the MANDATORY trace we're using.
	Trace Trace
}
