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
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/quic-go/quic-go"
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
func QUICHandshake(rt Runtime, options ...QUICHandshakeOption) Func[
	*Endpoint, *Maybe[*QUICConnection]] {
	// See https://github.com/ooni/probe/issues/2413 to understand
	// why we're using nil to force netxlite to use the cached
	// default Mozilla cert pool.
	f := &quicHandshakeFunc{
		InsecureSkipVerify: false,
		RootCAs:            nil,
		Rt:                 rt,
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

	// RootCAs contains the Root CAs to use.
	RootCAs *x509.CertPool

	// Rt is the Runtime that owns us.
	Rt Runtime

	// ServerName is the ServerName to handshake for.
	ServerName string

	dialer model.QUICDialer // for testing
}

// Apply implements Func.
func (f *quicHandshakeFunc) Apply(
	ctx context.Context, input *Endpoint) *Maybe[*QUICConnection] {
	// create trace
	trace := f.Rt.NewTrace(f.Rt.IDGenerator().Add(1), f.Rt.ZeroTime(), input.Tags...)

	// use defaults or user-configured overrides
	serverName := f.serverName(input)

	// start the operation logger
	ol := logx.NewOperationLogger(
		f.Rt.Logger(),
		"[#%d] QUICHandshake with %s SNI=%s",
		trace.Index(),
		input.Address,
		serverName,
	)

	// setup
	udpListener := netxlite.NewUDPListener()
	quicDialer := f.dialer
	if quicDialer == nil {
		quicDialer = trace.NewQUICDialerWithoutResolver(udpListener, f.Rt.Logger())
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
	f.Rt.Logger().Warn("TLSHandshake: cannot determine which SNI to use")
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
