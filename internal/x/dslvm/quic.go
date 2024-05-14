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
	"github.com/quic-go/quic-go"
)

// QUICHandshakeStage is a [Stage] that creates [*QUICConnection].
type QUICHandshakeStage struct {
	// Input contains the MANDATORY channel from which to read endpoints. We
	// assume that this channel will be closed when done.
	Input <-chan string

	// InsecureSkipVerify OPTIONALLY skips QUIC verification.
	InsecureSkipVerify bool

	// NextProtos OPTIONALLY configures the ALPN.
	NextProtos []string

	// Output is the MANDATORY channel emitting [*QUICConnection]. We will close this
	// channel when the Input channel has been closed.
	Output chan<- *QUICConnection

	// RootCAs OPTIONALLY configures alternative root CAs.
	RootCAs *x509.CertPool

	// ServerName is the MANDATORY server name.
	ServerName string

	// Tags contains OPTIONAL tags to add to the endpoint observations.
	Tags []string
}

// QUICConnection is a QUIC connection.
type QUICConnection struct {
	Conn      quic.EarlyConnection
	tlsConfig *tls.Config
	tx        Trace
}

// AsSingleUseTransport implements HTTPConnection.
func (c *QUICConnection) AsSingleUseTransport(logger model.Logger) model.HTTPTransport {
	return netxlite.NewHTTP3Transport(logger, netxlite.NewSingleUseQUICDialer(c.Conn), c.tlsConfig.Clone())
}

// Close implements HTTPConnection.
func (c *QUICConnection) Close(logger model.Logger) error {
	ol := logx.NewOperationLogger(logger, "[#%d] QUICClose %s", c.tx.Index(), c.RemoteAddress())
	err := c.Conn.CloseWithError(0, "")
	ol.Stop(err)
	return err
}

// Network implements HTTPConnection.
func (c *QUICConnection) Network() string {
	return "udp"
}

// RemoteAddress implements HTTPConnection.
func (c *QUICConnection) RemoteAddress() (addr string) {
	if v := c.Conn.RemoteAddr(); v != nil {
		addr = v.String()
	}
	return
}

// Scheme implements HTTPConnection.
func (c *QUICConnection) Scheme() string {
	return "https"
}

// TLSNegotiatedProtocol implements HTTPConnection.
func (c *QUICConnection) TLSNegotiatedProtocol() string {
	return c.Conn.ConnectionState().TLS.NegotiatedProtocol
}

// Trace implements HTTPConnection.
func (c *QUICConnection) Trace() Trace {
	return c.tx
}

var _ HTTPConnection = &QUICConnection{}

// Run is like [*TCPConnect.Run] except that it reads [endpoints] in Input and
// emits [*QUICConnection] in Output. Each QUIC handshake runs in its own background
// goroutine. The parallelism is controlled by the [Runtime] ActiveConnections [Semaphore] and
// you MUST arrange for the [*QUICConnection] to eventually enter into a [*CloseStage]
// such that the code can release the above mentioned [Semaphore] and close the conn. Note
// that this code TAKES OWNERSHIP of the [*TCPConnection] it reads. We will close these
// conns automatically on failure. On success, they will be closed when the [*QUICConnection]
// wrapping them eventually enters into a [*CloseStage].
func (sx *QUICHandshakeStage) Run(ctx context.Context, rtx Runtime) {
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
			sx.handshake(ctx, rtx, endpoint)
		}(endpoint)
	}

	// wait for pending work to finish
	waitGroup.Wait()
}

func (sx *QUICHandshakeStage) handshake(ctx context.Context, rtx Runtime, endpoint string) {
	// create trace
	trace := rtx.NewTrace(rtx.IDGenerator().Add(1), rtx.ZeroTime(), sx.Tags...)

	// create a suitable QUIC configuration
	config := sx.newTLSConfig()

	// start the operation logger
	ol := logx.NewOperationLogger(
		rtx.Logger(),
		"[#%d] QUICHandshake with %s SNI=%s ALPN=%v",
		trace.Index(),
		endpoint,
		config.ServerName,
		config.NextProtos,
	)

	// setup
	udpListener := trace.NewUDPListener()
	quicDialer := trace.NewQUICDialerWithoutResolver(udpListener, rtx.Logger())
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// handshake
	quicConn, err := quicDialer.DialContext(ctx, endpoint, config, &quic.Config{})

	// stop the operation logger
	ol.Stop(err)

	// save the observations
	rtx.SaveObservations(maybeTraceToObservations(trace)...)

	// handle error case
	if err != nil {
		return
	}

	// TODO(https://github.com/ooni/probe/issues/2670).
	//
	// Start measuring for throttling here.

	// handle success
	sx.Output <- &QUICConnection{Conn: quicConn, tx: trace, tlsConfig: config}
}

func (sx *QUICHandshakeStage) newTLSConfig() *tls.Config {
	return &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		NextProtos:         sx.NextProtos,
		InsecureSkipVerify: sx.InsecureSkipVerify,
		RootCAs:            sx.RootCAs,
		ServerName:         sx.ServerName,
	}
}
