package dslvm

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TCPConnectStage is a [Stage] that creates [*TCPConnection].
type TCPConnectStage struct {
	// Input contains the MANDATORY channel from which to read endpoints. We
	// assume that this channel will be closed when done.
	Input <-chan string

	// Output is the MANDATORY channel emitting [*TCPConnection]. We will close this
	// channel when the Input channel has been closed.
	Output chan<- *TCPConnection

	// Tags contains OPTIONAL tags to add to the endpoint observations.
	Tags []string
}

// TCPConnection is a TCP connection.
type TCPConnection struct {
	Conn net.Conn
	tx   Trace
}

var _ HTTPConnection = &TCPConnection{}

// AsSingleUseTransport implements HTTPConnection.
func (c *TCPConnection) AsSingleUseTransport(logger model.Logger) model.HTTPTransport {
	return netxlite.NewHTTPTransport(logger, netxlite.NewSingleUseDialer(c.Conn), netxlite.NewNullTLSDialer())
}

// Close implements HTTPConnection.
func (c *TCPConnection) Close(logger model.Logger) error {
	ol := logx.NewOperationLogger(logger, "[#%d] TCPClose %s", c.tx.Index(), c.RemoteAddress())
	err := c.Conn.Close()
	ol.Stop(err)
	return err
}

// Network implements HTTPConnection.
func (c *TCPConnection) Network() string {
	return "tcp"
}

// RemoteAddress implements HTTPConnection.
func (c *TCPConnection) RemoteAddress() (addr string) {
	if v := c.Conn.RemoteAddr(); v != nil {
		addr = v.String()
	}
	return
}

// Scheme implements HTTPConnection.
func (c *TCPConnection) Scheme() string {
	return "http"
}

// TLSNegotiatedProtocol implements HTTPConnection.
func (c *TCPConnection) TLSNegotiatedProtocol() string {
	return ""
}

// Trace implements HTTPConnection.
func (c *TCPConnection) Trace() Trace {
	return c.tx
}

var _ Stage = &TCPConnectStage{}

// Run reads endpoints from Input and streams on the Output channel the [*TCPConnection]
// that it could successfully establish. Note that this function honors the [Semaphore] returned
// by the [Runtime] ActiveConnections that controls how many connections we can measure in
// parallel. When given the permission to run, this function spawns a background goroutine that
// attempts to establish a connection. The [*TCPConnection] returned by this stage MUST
// eventually feed into a [*CloseStage], so that the code can notify the above mentioned
// [Semaphore] and so that we close the open connection. This function will close the Output
// channel when Inputs have been closed and there are no pending connection attempts. In
// case of failure, the code will automatically notify the [Semaphore].
func (sx *TCPConnectStage) Run(ctx context.Context, rtx Runtime) {
	// make sure we close the output channel
	defer close(sx.Output)

	// track the number of running goroutines
	waitGroup := &sync.WaitGroup{}

	for endpoint := range sx.Input {
		// wait for authorization to process a connection
		rtx.ActiveConnections().Wait()

		// process connection in a background goroutine
		waitGroup.Add(1)
		go func(endpoint string) {
			defer waitGroup.Done()
			sx.connect(ctx, rtx, endpoint)
		}(endpoint)
	}

	// wait for pending work to finish
	waitGroup.Wait()
}

func (sx *TCPConnectStage) connect(ctx context.Context, rtx Runtime, endpoint string) {
	// create trace
	trace := rtx.NewTrace(rtx.IDGenerator().Add(1), rtx.ZeroTime(), sx.Tags...)

	// start operation logger
	ol := logx.NewOperationLogger(
		rtx.Logger(),
		"[#%d] TCPConnect %s",
		trace.Index(),
		endpoint,
	)

	// setup
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// obtain the dialer to use
	dialer := trace.NewDialerWithoutResolver(rtx.Logger())

	// connect
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)

	// stop the operation logger
	ol.Stop(err)

	// save the observations
	rtx.SaveObservations(maybeTraceToObservations(trace)...)

	// handle error case
	if err != nil {
		rtx.ActiveConnections().Signal() // make sure we release the semaphore
		return
	}

	// handle success
	sx.Output <- &TCPConnection{Conn: conn, tx: trace}
}
