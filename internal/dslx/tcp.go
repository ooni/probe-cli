package dslx

//
// TCP measurements
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TCPConnect returns a function that establishes TCP connections.
func TCPConnect(rt Runtime) Func[*Endpoint, *Maybe[*TCPConnection]] {
	f := &tcpConnectFunc{rt}
	return f
}

// tcpConnectFunc is a function that establishes TCP connections.
type tcpConnectFunc struct {
	rt Runtime
}

// Apply applies the function to its arguments.
func (f *tcpConnectFunc) Apply(
	ctx context.Context, input *Endpoint) *Maybe[*TCPConnection] {

	// create trace
	trace := f.rt.NewTrace(f.rt.IDGenerator().Add(1), f.rt.ZeroTime(), input.Tags...)

	// start the operation logger
	ol := logx.NewOperationLogger(
		f.rt.Logger(),
		"[#%d] TCPConnect %s",
		trace.Index(),
		input.Address,
	)

	// setup
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// obtain the dialer to use
	dialer := trace.NewDialerWithoutResolver(f.rt.Logger())

	// connect
	conn, err := dialer.DialContext(ctx, "tcp", input.Address)

	// possibly register established conn for late close
	f.rt.MaybeTrackConn(conn)

	// stop the operation logger
	ol.Stop(err)

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
		Operation:    netxlite.ConnectOperation,
		State:        state,
	}
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
