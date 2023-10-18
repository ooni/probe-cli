package dslx

//
// TLS measurements
//

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TLSHandshakeOption is an option you can pass to TLSHandshake.
type TLSHandshakeOption func(config *tls.Config)

// TLSHandshakeOptionInsecureSkipVerify controls whether TLS verification is enabled.
func TLSHandshakeOptionInsecureSkipVerify(value bool) TLSHandshakeOption {
	return func(config *tls.Config) {
		config.InsecureSkipVerify = value
	}
}

// TLSHandshakeOptionNextProto allows to configure the ALPN protocols.
func TLSHandshakeOptionNextProto(value []string) TLSHandshakeOption {
	return func(config *tls.Config) {
		config.NextProtos = value
	}
}

// TLSHandshakeOptionRootCAs allows to configure custom root CAs.
func TLSHandshakeOptionRootCAs(value *x509.CertPool) TLSHandshakeOption {
	return func(config *tls.Config) {
		config.RootCAs = value
	}
}

// TLSHandshakeOptionServerName allows to configure the SNI to use.
func TLSHandshakeOptionServerName(value string) TLSHandshakeOption {
	return func(config *tls.Config) {
		config.ServerName = value
	}
}

// TLSHandshake returns a function performing TSL handshakes.
func TLSHandshake(rt Runtime, options ...TLSHandshakeOption) Func[
	*TCPConnection, *Maybe[*TLSConnection]] {
	f := &tlsHandshakeFunc{
		Options: options,
		Rt:      rt,
	}
	return f
}

// tlsHandshakeFunc performs TLS handshakes.
type tlsHandshakeFunc struct {
	// Options contains the options.
	Options []TLSHandshakeOption

	// Rt is the runtime that owns us.
	Rt Runtime
}

// Apply implements Func.
func (f *tlsHandshakeFunc) Apply(
	ctx context.Context, input *TCPConnection) *Maybe[*TLSConnection] {
	// keep using the same trace
	trace := input.Trace

	// create a suitable TLS configuration
	config := tlsNewConfig(input.Address, []string{"h2", "http/1.1"}, input.Domain, f.Rt.Logger(), f.Options...)

	// start the operation logger
	ol := logx.NewOperationLogger(
		f.Rt.Logger(),
		"[#%d] TLSHandshake with %s SNI=%s ALPN=%v",
		trace.Index(),
		input.Address,
		config.ServerName,
		config.NextProtos,
	)

	// obtain the handshaker for use
	handshaker := trace.NewTLSHandshakerStdlib(f.Rt.Logger())

	// setup
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// handshake
	conn, err := handshaker.Handshake(ctx, input.Conn, config)

	// possibly register established conn for late close
	f.Rt.MaybeTrackConn(conn)

	// stop the operation logger
	ol.Stop(err)

	state := &TLSConnection{
		Address:  input.Address,
		Conn:     conn, // possibly nil
		Domain:   input.Domain,
		Network:  input.Network,
		TLSState: netxlite.MaybeTLSConnectionState(conn),
		Trace:    trace,
	}

	return &Maybe[*TLSConnection]{
		Error:        err,
		Observations: maybeTraceToObservations(trace),
		Operation:    netxlite.TLSHandshakeOperation,
		State:        state,
	}
}

// tlsNewConfig is an utility function to create a new TLS config.
//
// Arguments:
//
// - address is the endpoint address (e.g., 1.1.1.1:443);
//
// - defaultALPN contains the default to be used for configuring ALPN;
//
// - domain is the possibly empty domain to use;
//
// - logger is the logger to use;
//
// - options contains options to modify the TLS handshake defaults.
func tlsNewConfig(address string, defaultALPN []string, domain string, logger model.Logger, options ...TLSHandshakeOption) *tls.Config {
	// See https://github.com/ooni/probe/issues/2413 to understand
	// why we're using nil to force netxlite to use the cached
	// default Mozilla cert pool.
	config := &tls.Config{
		NextProtos:         append([]string{}, defaultALPN...),
		InsecureSkipVerify: false,
		RootCAs:            nil,
		ServerName:         tlsServerName(address, domain, logger),
	}
	for _, option := range options {
		option(config)
	}
	return config
}

// tlsServerName is an utility function to obtina the server name from a TCPConnection.
func tlsServerName(address, domain string, logger model.Logger) string {
	if domain != "" {
		return domain
	}
	addr, _, err := net.SplitHostPort(address)
	if err == nil {
		return addr
	}
	// Note: golang requires a ServerName and fails if it's empty. If the provided
	// ServerName is an IP address, however, golang WILL NOT emit any SNI extension
	// in the ClientHello, consistently with RFC 6066 Section 3 requirements.
	logger.Warn("TLSHandshake: cannot determine which SNI to use")
	return ""
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

	// Network is the MANDATORY network we tried to use when connecting.
	Network string

	// TLSState is the possibly-empty TLS connection state.
	TLSState tls.ConnectionState

	// Trace is the MANDATORY trace we're using.
	Trace Trace
}
