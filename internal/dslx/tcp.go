package dslx

//
// TCP measurements
//

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TCPConnect returns a function that establishes TCP connections.
func TCPConnect(pool *ConnPool) Func[*Endpoint, *Maybe[*TCPConnection]] {
	f := &tcpConnectFunc{pool, nil}
	return f
}

// tcpConnectFunc is a function that establishes TCP connections.
type tcpConnectFunc struct {
	p      *ConnPool
	dialer model.Dialer // for testing
}

// Apply applies the function to its arguments.
func (f *tcpConnectFunc) Apply(
	ctx context.Context, input *Endpoint) *Maybe[*TCPConnection] {

	// create trace
	trace := measurexlite.NewTrace(input.IDGenerator.Add(1), input.ZeroTime, input.Tags...)

	// start the operation logger
	ol := logx.NewOperationLogger(
		input.Logger,
		"[#%d] TCPConnect %s",
		trace.Index,
		input.Address,
	)

	// setup
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// obtain the dialer to use
	dialer := f.dialerOrDefault(trace, input.Logger)

	// connect
	conn, err := dialer.DialContext(ctx, "tcp", input.Address)

	// possibly register established conn for late close
	f.p.MaybeTrack(conn)

	// stop the operation logger
	ol.Stop(err)

	state := &TCPConnection{
		Address:     input.Address,
		Conn:        conn, // possibly nil
		Domain:      input.Domain,
		IDGenerator: input.IDGenerator,
		Logger:      input.Logger,
		Network:     input.Network,
		Trace:       trace,
		ZeroTime:    input.ZeroTime,
	}

	return &Maybe[*TCPConnection]{
		Error:        err,
		Observations: maybeTraceToObservations(trace),
		Operation:    netxlite.ConnectOperation,
		State:        state,
	}
}

// dialerOrDefault is the function used to obtain a dialer
func (f *tcpConnectFunc) dialerOrDefault(trace *measurexlite.Trace, logger model.Logger) model.Dialer {
	dialer := f.dialer
	if dialer == nil {
		dialer = trace.NewDialerWithoutResolver(logger)
	}
	return dialer
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

	// IDGenerator is the MANDATORY ID generator.
	IDGenerator *atomic.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// Network is the MANDATORY network we tried to use when connecting.
	Network string

	// Trace is the MANDATORY trace we're using.
	Trace *measurexlite.Trace

	// ZeroTime is the MANDATORY zero time of the measurement.
	ZeroTime time.Time
}
