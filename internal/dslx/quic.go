package dslx

//
// QUIC measurements
//

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// QUICHandshakeOption is an option you can pass to QUICHandshake.
type QUICHandshakeOption func(*quicHandshakeFunc)

// QUICHandshakeOptionInsecureSkipVerify controls whether QUIC verification is enabled.
func QUICHandshakeOptionInsecureSkipVerify(value bool) QUICHandshakeOption {
	return func(thf *quicHandshakeFunc) {
		thf.InsecureSkipVerify = value
	}
}

// QUICHandshakeOptionRootCAs allows to configure custom root CAs.
func QUICHandshakeOptionRootCAs(value *x509.CertPool) QUICHandshakeOption {
	return func(thf *quicHandshakeFunc) {
		thf.RootCAs = value
	}
}

// QUICHandshakeOptionServerName allows to configure the SNI to use.
func QUICHandshakeOptionServerName(value string) QUICHandshakeOption {
	return func(thf *quicHandshakeFunc) {
		thf.ServerName = value
	}
}

// QUICHandshake returns a function performing QUIC handshakes.
func QUICHandshake(pool *ConnPool, options ...QUICHandshakeOption) Func[
	*Endpoint, *Maybe[*QUICConnection]] {
	f := &quicHandshakeFunc{
		InsecureSkipVerify: false,
		Pool:               pool,
		ServerName:         "",
	}
	for _, option := range options {
		option(f)
	}
	return f
}

// quicHandshakeFunc performs QUIC handshakes.
type quicHandshakeFunc struct {
	// InsecureSkipVerify allows to skip TLS verification.
	InsecureSkipVerify bool

	// Pool is the ConnPool that owns us.
	Pool *ConnPool

	// RootCAs contains the Root CAs to use.
	RootCAs *x509.CertPool

	// ServerName is the ServerName to handshake for.
	ServerName string

	dialer model.QUICDialer // for testing
}

// Apply implements Func.
func (f *quicHandshakeFunc) Apply(
	ctx context.Context, input *Endpoint) *Maybe[*QUICConnection] {
	// create trace
	trace := measurexlite.NewTrace(input.IDGenerator.Add(1), input.ZeroTime)

	// use defaults or user-configured overrides
	serverName := f.serverName(input)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(
		input.Logger,
		"[#%d] QUICHandshake with %s SNI=%s",
		trace.Index,
		input.Address,
		serverName,
	)

	// setup
	quicListener := netxlite.NewQUICListener()
	quicDialer := f.dialer
	if quicDialer == nil {
		quicDialer = trace.NewQUICDialerWithoutResolver(quicListener, input.Logger)
	}
	config := &tls.Config{
		NextProtos:         []string{"h3"},
		InsecureSkipVerify: f.InsecureSkipVerify,
		RootCAs:            f.RootCAs,
		ServerName:         serverName,
	}
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// handshake
	quicConn, err := quicDialer.DialContext(ctx, input.Address, config, &quic.Config{})

	var closerConn io.Closer
	var tlsState tls.ConnectionState
	if quicConn != nil {
		closerConn = &quicCloserConn{quicConn}
		tlsState = quicConn.ConnectionState().TLS.ConnectionState // only quicConn can be nil
	}

	// possibly track established conn for late close
	f.Pool.MaybeTrack(closerConn)

	// stop the operation logger
	ol.Stop(err)

	state := &QUICConnection{
		Address:     input.Address,
		QUICConn:    quicConn, // possibly nil
		Domain:      input.Domain,
		IDGenerator: input.IDGenerator,
		Logger:      input.Logger,
		Network:     input.Network,
		TLSConfig:   config,
		TLSState:    tlsState,
		Trace:       trace,
		ZeroTime:    input.ZeroTime,
	}

	return &Maybe[*QUICConnection]{
		Error:        err,
		Observations: maybeTraceToObservations(trace),
		Operation:    netxlite.QUICHandshakeOperation,
		State:        state,
	}
}

func (f *quicHandshakeFunc) serverName(input *Endpoint) string {
	if f.ServerName != "" {
		return f.ServerName
	}
	if input.Domain != "" {
		return input.Domain
	}
	addr, _, err := net.SplitHostPort(input.Address)
	if err == nil {
		return addr
	}
	// Note: golang requires a ServerName and fails if it's empty. If the provided
	// ServerName is an IP address, however, golang WILL NOT emit any SNI extension
	// in the ClientHello, consistently with RFC 6066 Section 3 requirements.
	input.Logger.Warn("TLSHandshake: cannot determine which SNI to use")
	return ""
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

	// IDGenerator is the MANDATORY ID generator to use.
	IDGenerator *atomic.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// Network is the MANDATORY network we tried to use when connecting.
	Network string

	// TLSConfig is the config we used to establish a QUIC connection and will
	// be needed when constructing an HTTP/3 transport.
	TLSConfig *tls.Config

	// TLSState is the possibly-empty TLS connection state.
	TLSState tls.ConnectionState

	// Trace is the MANDATORY trace we're using.
	Trace *measurexlite.Trace

	// ZeroTime is the MANDATORY zero time of the measurement.
	ZeroTime time.Time
}

type quicCloserConn struct {
	quic.EarlyConnection
}

func (c *quicCloserConn) Close() error {
	return c.CloseWithError(0, "")
}
