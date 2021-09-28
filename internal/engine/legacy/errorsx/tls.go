package errorsx

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TLSHandshaker is the generic TLS handshaker
type TLSHandshaker interface {
	Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}

// ErrorWrapperTLSHandshaker wraps the returned error to be an OONI error
type ErrorWrapperTLSHandshaker struct {
	TLSHandshaker
}

// Handshake implements TLSHandshaker.Handshake
func (h *ErrorWrapperTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	err = SafeErrWrapperBuilder{
		Classifier: netxlite.ClassifyTLSHandshakeError,
		Error:      err,
		Operation:  netxlite.TLSHandshakeOperation,
	}.MaybeBuild()
	return tlsconn, state, err
}
