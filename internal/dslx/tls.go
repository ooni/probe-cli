package dslx

//
// TLS measurements
//

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TLSHandshakeOption is an option you can pass to TLSHandshake.
type TLSHandshakeOption func(*tlsHandshakeFunc)

// TLSHandshakeOptionInsecureSkipVerify controls whether TLS verification is enabled.
func TLSHandshakeOptionInsecureSkipVerify(value bool) TLSHandshakeOption {
	return func(thf *tlsHandshakeFunc) {
		thf.InsecureSkipVerify = value
	}
}

// TLSHandshakeOptionNextProto allows to configure the ALPN protocols.
func TLSHandshakeOptionNextProto(value []string) TLSHandshakeOption {
	return func(thf *tlsHandshakeFunc) {
		thf.NextProto = value
	}
}

// TLSHandshakeOptionRootCAs allows to configure custom root CAs.
func TLSHandshakeOptionRootCAs(value *x509.CertPool) TLSHandshakeOption {
	return func(thf *tlsHandshakeFunc) {
		thf.RootCAs = value
	}
}

// TLSHandshakeOptionServerName allows to configure the SNI to use.
func TLSHandshakeOptionServerName(value string) TLSHandshakeOption {
	return func(thf *tlsHandshakeFunc) {
		thf.ServerName = value
	}
}

// TLSHandshake returns a function performing TSL handshakes.
func TLSHandshake(pool *ConnPool, options ...TLSHandshakeOption) Func[
	*TCPConnection, *Maybe[*TLSConnection]] {
	f := &tlsHandshakeFunc{
		InsecureSkipVerify: false,
		NextProto:          []string{},
		Pool:               pool,
		ServerName:         "",
	}
	for _, option := range options {
		option(f)
	}
	return f
}

// tlsHandshakeFunc performs TLS handshakes.
type tlsHandshakeFunc struct {
	// InsecureSkipVerify allows to skip TLS verification.
	InsecureSkipVerify bool

	// NextProto contains the ALPNs to negotiate.
	NextProto []string

	// Pool is the Pool that owns us.
	Pool *ConnPool

	// RootCAs contains the Root CAs to use.
	RootCAs *x509.CertPool

	// ServerName is the ServerName to handshake for.
	ServerName string

	// for testing
	handshaker model.TLSHandshaker
}

// Apply implements Func.
func (f *tlsHandshakeFunc) Apply(
	ctx context.Context, input *TCPConnection) *Maybe[*TLSConnection] {
	// keep using the same trace
	trace := input.Trace

	// use defaults or user-configured overrides
	serverName := f.serverName(input)
	nextProto := f.nextProto()

	// start the operation logger
	ol := measurexlite.NewOperationLogger(
		input.Logger,
		"[#%d] TLSHandshake with %s SNI=%s ALPN=%v",
		trace.Index,
		input.Address,
		serverName,
		nextProto,
	)

	// setup
	handshaker := f.handshaker
	if handshaker == nil {
		handshaker = trace.NewTLSHandshakerStdlib(input.Logger)
	}
	config := &tls.Config{
		NextProtos:         nextProto,
		InsecureSkipVerify: f.InsecureSkipVerify,
		RootCAs:            f.RootCAs,
		ServerName:         serverName,
	}
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// handshake
	conn, tlsState, err := handshaker.Handshake(ctx, input.Conn, config)

	// possibly register established conn for late close
	f.Pool.MaybeTrack(conn)

	// stop the operation logger
	ol.Stop(err)

	var tlsConn netxlite.TLSConn
	if conn != nil {
		tlsConn = conn.(netxlite.TLSConn) // guaranteed to work
	}

	state := &TLSConnection{
		Address:     input.Address,
		Conn:        tlsConn, // possibly nil
		Domain:      input.Domain,
		IDGenerator: input.IDGenerator,
		Logger:      input.Logger,
		Network:     input.Network,
		TLSState:    tlsState,
		Trace:       trace,
		ZeroTime:    input.ZeroTime,
	}

	return &Maybe[*TLSConnection]{
		Error:        err,
		Observations: maybeTraceToObservations(trace),
		Operation:    netxlite.TLSHandshakeOperation,
		State:        state,
	}
}

func (f *tlsHandshakeFunc) serverName(input *TCPConnection) string {
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

func (f *tlsHandshakeFunc) nextProto() []string {
	if len(f.NextProto) > 0 {
		return f.NextProto
	}
	return []string{"h2", "http/1.1"}
}

// TLSConnection is an established TLS connection. If you initialize
// manually, init at least the ones marked as MANDATORY.
type TLSConnection struct {
	// Address is the MANDATORY address we tried to connect to.
	Address string

	// Conn is the established TLS conn.
	Conn netxlite.TLSConn

	// Domain is the OPTIONAL domain we resolved.
	Domain string

	// IDGenerator is the MANDATORY ID generator to use.
	IDGenerator *atomic.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// Network is the MANDATORY network we tried to use when connecting.
	Network string

	// TLSState is the possibly-empty TLS connection state.
	TLSState tls.ConnectionState

	// Trace is the MANDATORY trace we're using.
	Trace *measurexlite.Trace

	// ZeroTime is the MANDATORY zero time of the measurement.
	ZeroTime time.Time
}
