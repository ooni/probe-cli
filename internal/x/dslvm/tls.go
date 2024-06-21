package dslvm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TLSHandshakeStage is a [Stage] that creates [*TLSConnection].
type TLSHandshakeStage struct {
	// Input contains the MANDATORY channel from which to read [*TCPConnection]. We
	// assume that this channel will be closed when done.
	Input <-chan *TCPConnection

	// InsecureSkipVerify OPTIONALLY skips TLS verification.
	InsecureSkipVerify bool

	// NextProtos OPTIONALLY configures the ALPN.
	NextProtos []string

	// Output is the MANDATORY channel emitting [*TLSConnection]. We will close this
	// channel when the Input channel has been closed.
	Output chan<- *TLSConnection

	// RootCAs OPTIONALLY configures alternative root CAs.
	RootCAs *x509.CertPool

	// ServerName is the MANDATORY server name.
	ServerName string
}

// TLSConnection is a TLS connection.
type TLSConnection struct {
	Conn model.TLSConn
	tx   Trace
}

var _ HTTPConnection = &TLSConnection{}

// AsSingleUseTransport implements HTTPConnection.
func (c *TLSConnection) AsSingleUseTransport(logger model.Logger) model.HTTPTransport {
	return netxlite.NewHTTPTransport(logger, netxlite.NewNullDialer(), netxlite.NewSingleUseTLSDialer(c.Conn))
}

// Close implements HTTPConnection.
func (c *TLSConnection) Close(logger model.Logger) error {
	ol := logx.NewOperationLogger(logger, "[#%d] TLSClose %s", c.tx.Index(), c.RemoteAddress())
	err := c.Conn.Close()
	ol.Stop(err)
	return err
}

// Network implements HTTPConnection.
func (c *TLSConnection) Network() string {
	return "tcp"
}

// RemoteAddress implements HTTPConnection.
func (c *TLSConnection) RemoteAddress() (addr string) {
	if v := c.Conn.RemoteAddr(); v != nil {
		addr = v.String()
	}
	return
}

// Scheme implements HTTPConnection.
func (c *TLSConnection) Scheme() string {
	return "https"
}

// TLSNegotiatedProtocol implements HTTPConnection.
func (c *TLSConnection) TLSNegotiatedProtocol() string {
	return c.Conn.ConnectionState().NegotiatedProtocol
}

// Trace implements HTTPConnection.
func (c *TLSConnection) Trace() Trace {
	return c.tx
}

// Run is like [*TCPConnect.Run] except that it reads [*TCPConnection] in Input and
// emits [*TLSConnection] in Output. Each TLS handshake runs in its own background
// goroutine. The parallelism is controlled by the [Runtime] ActiveConnections [Semaphore]
// and you MUST arrange for the [*TLSConnection] to eventually enter into a [*CloseStage]
// such that the code can release the above mentioned [Semaphore] and close the conn. Note
// that this code TAKES OWNERSHIP of the [*TCPConnection] it reads. We will close these
// conns automatically on failure. On success, they will be closed when the [*TLSConnection]
// wrapping them eventually enters into a [*CloseStage].
func (sx *TLSHandshakeStage) Run(ctx context.Context, rtx Runtime) {
	// make sure we close the output channel
	defer close(sx.Output)

	// track the number of running goroutines
	waitGroup := &sync.WaitGroup{}

	for tcpConn := range sx.Input {
		// process connection in a background goroutine, which is fine
		// because the previous step has acquired the semaphore.
		waitGroup.Add(1)
		go func(tcpConn *TCPConnection) {
			defer waitGroup.Done()
			sx.handshake(ctx, rtx, tcpConn)
		}(tcpConn)
	}

	// wait for pending work to finish
	waitGroup.Wait()
}

func (sx *TLSHandshakeStage) handshake(ctx context.Context, rtx Runtime, tcpConn *TCPConnection) {
	// keep using the same trace
	trace := tcpConn.Trace()

	// create a suitable TLS configuration
	config := sx.newTLSConfig()

	// start the operation logger
	ol := logx.NewOperationLogger(
		rtx.Logger(),
		"[#%d] TLSHandshake with %s SNI=%s ALPN=%v",
		trace.Index(),
		tcpConn.RemoteAddress(),
		config.ServerName,
		config.NextProtos,
	)

	// obtain the handshaker for use
	handshaker := trace.NewTLSHandshakerStdlib(rtx.Logger())

	// setup
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// handshake
	tlsConn, err := handshaker.Handshake(ctx, tcpConn.Conn, config)

	// stop the operation logger
	ol.Stop(err)

	// save the observations
	rtx.SaveObservations(maybeTraceToObservations(trace)...)

	// handle error case
	if err != nil {
		rtx.ActiveConnections().Signal() // make sure we release the semaphore
		_ = tcpConn.Conn.Close()         // make sure we close the conn
		return
	}

	// handle success
	sx.Output <- &TLSConnection{
		Conn: tlsConn,
		tx:   trace,
	}
}

func (sx *TLSHandshakeStage) newTLSConfig() *tls.Config {
	return &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		NextProtos:         sx.NextProtos,
		InsecureSkipVerify: sx.InsecureSkipVerify, // #nosec G402 - it's fine to possibly skip verify in a nettest
		RootCAs:            sx.RootCAs,
		ServerName:         sx.ServerName,
	}
}
