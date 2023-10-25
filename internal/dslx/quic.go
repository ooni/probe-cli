package dslx

//
// QUIC measurements
//

import (
	"context"
	"crypto/tls"
	"io"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/quic-go/quic-go"
)

// QUICHandshake returns a function performing QUIC handshakes.
func QUICHandshake(rt Runtime, options ...TLSHandshakeOption) Func[
	*Endpoint, *Maybe[*QUICConnection]] {
	f := &quicHandshakeFunc{
		Options: options,
		Rt:      rt,
	}
	return f
}

// quicHandshakeFunc performs QUIC handshakes.
type quicHandshakeFunc struct {
	// Options contains the options.
	Options []TLSHandshakeOption

	// Rt is the Runtime that owns us.
	Rt Runtime
}

// Apply implements Func.
func (f *quicHandshakeFunc) Apply(
	ctx context.Context, input *Endpoint) *Maybe[*QUICConnection] {
	// create trace
	trace := f.Rt.NewTrace(f.Rt.IDGenerator().Add(1), f.Rt.ZeroTime(), input.Tags...)

	// create a suitable TLS configuration
	config := tlsNewConfig(input.Address, []string{"h3"}, input.Domain, f.Rt.Logger(), f.Options...)

	// start the operation logger
	ol := logx.NewOperationLogger(
		f.Rt.Logger(),
		"[#%d] QUICHandshake with %s SNI=%s ALPN=%v",
		trace.Index(),
		input.Address,
		config.ServerName,
		config.NextProtos,
	)

	// setup
	udpListener := netxlite.NewUDPListener()
	quicDialer := trace.NewQUICDialerWithoutResolver(udpListener, f.Rt.Logger())
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// handshake
	quicConn, err := quicDialer.DialContext(ctx, input.Address, config, &quic.Config{})

	var closerConn io.Closer
	var tlsState tls.ConnectionState
	if quicConn != nil {
		closerConn = &quicCloserConn{quicConn}
		tlsState = quicConn.ConnectionState().TLS // only quicConn can be nil
	}

	// possibly track established conn for late close
	f.Rt.MaybeTrackConn(closerConn)

	// stop the operation logger
	ol.Stop(err)

	state := &QUICConnection{
		Address:   input.Address,
		QUICConn:  quicConn, // possibly nil
		Domain:    input.Domain,
		Network:   input.Network,
		TLSConfig: config,
		TLSState:  tlsState,
		Trace:     trace,
	}

	return &Maybe[*QUICConnection]{
		Error:        err,
		Observations: maybeTraceToObservations(trace),
		Operation:    netxlite.QUICHandshakeOperation,
		State:        state,
	}
}

// QUICConnection is an established QUIC connection. If you initialize
// manually, init at least the ones marked as MANDATORY.
type QUICConnection struct {
	// Address is the MANDATORY address we tried to connect to.
	Address string

	// QUICConn is the established QUIC conn.
	QUICConn quic.EarlyConnection

	// Domain is the OPTIONAL domain we resolved.
	Domain string

	// Network is the MANDATORY network we tried to use when connecting.
	Network string

	// TLSConfig is the config we used to establish a QUIC connection and will
	// be needed when constructing an HTTP/3 transport.
	TLSConfig *tls.Config

	// TLSState is the possibly-empty TLS connection state.
	TLSState tls.ConnectionState

	// Trace is the MANDATORY trace we're using.
	Trace Trace
}

type quicCloserConn struct {
	quic.EarlyConnection
}

func (c *quicCloserConn) Close() error {
	return c.CloseWithError(0, "")
}
