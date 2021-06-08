package tlsdialer

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsx"
)

// Logger is the logger assumed by this package
type Logger interface {
	Debugf(format string, v ...interface{})
	Debug(message string)
}

// LoggingTLSHandshaker is a TLSHandshaker with logging
type LoggingTLSHandshaker struct {
	TLSHandshaker
	Logger Logger
}

// Handshake implements Handshaker.Handshake
func (h LoggingTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	h.Logger.Debugf("tls {sni=%s next=%+v}...", config.ServerName, config.NextProtos)
	start := time.Now()
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	stop := time.Now()
	h.Logger.Debugf(
		"tls {sni=%s next=%+v}... %+v in %s {next=%s cipher=%s v=%s}", config.ServerName,
		config.NextProtos, err, stop.Sub(start), state.NegotiatedProtocol,
		tlsx.CipherSuiteString(state.CipherSuite), tlsx.VersionString(state.Version))
	return tlsconn, state, err
}

var _ TLSHandshaker = LoggingTLSHandshaker{}
