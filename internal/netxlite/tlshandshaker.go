package netxlite

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsx"
)

// TLSHandshaker is the generic TLS handshaker.
type TLSHandshaker interface {
	// Handshake creates a new TLS connection from the given connection and
	// the given config. This function DOES NOT take ownership of the connection
	// and it's your responsibility to close it on failure.
	Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}

// TLSHandshakerStdlib is the stdlib's TLS handshaker.
type TLSHandshakerStdlib struct {
	// Timeout is the timeout imposed on the TLS handshake. If zero
	// or negative, we will use default timeout of 10 seconds.
	Timeout time.Duration
}

var _ TLSHandshaker = &TLSHandshakerStdlib{}

// Handshake implements Handshaker.Handshake
func (h *TLSHandshakerStdlib) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	defer conn.SetDeadline(time.Time{})
	conn.SetDeadline(time.Now().Add(timeout))
	tlsconn := tls.Client(conn, config)
	if err := tlsconn.Handshake(); err != nil {
		return nil, tls.ConnectionState{}, err
	}
	return tlsconn, tlsconn.ConnectionState(), nil
}

// DefaultTLSHandshaker is the default TLS handshaker.
var DefaultTLSHandshaker = &TLSHandshakerStdlib{}

// TLSHandshakerLogger is a TLSHandshaker with logging.
type TLSHandshakerLogger struct {
	// TLSHandshaker is the underlying handshaker.
	TLSHandshaker TLSHandshaker

	// Logger is the underlying logger.
	Logger Logger
}

// Handshake implements Handshaker.Handshake
func (h *TLSHandshakerLogger) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	h.Logger.Debugf(
		"tls {sni=%s next=%+v}...", config.ServerName, config.NextProtos)
	start := time.Now()
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	elapsed := time.Since(start)
	if err != nil {
		h.Logger.Debugf(
			"tls {sni=%s next=%+v}... %s in %s", config.ServerName,
			config.NextProtos, err, elapsed)
		return nil, tls.ConnectionState{}, err
	}
	h.Logger.Debugf(
		"tls {sni=%s next=%+v}... ok in %s {next=%s cipher=%s v=%s}",
		config.ServerName, config.NextProtos, elapsed, state.NegotiatedProtocol,
		tlsx.CipherSuiteString(state.CipherSuite),
		tlsx.VersionString(state.Version))
	return tlsconn, state, nil
}

var _ TLSHandshaker = &TLSHandshakerLogger{}
