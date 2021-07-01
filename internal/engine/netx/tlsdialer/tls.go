// Package tlsdialer contains code to establish TLS connections.
package tlsdialer

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

// UnderlyingDialer is the underlying dialer type.
type UnderlyingDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// TLSHandshaker is the generic TLS handshaker
type TLSHandshaker interface {
	Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}

// ErrorWrapperTLSHandshaker wraps the returned error to be an OONI error
type ErrorWrapperTLSHandshaker struct {
	TLSHandshaker
}

// Handshake implements Handshaker.Handshake
func (h ErrorWrapperTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	err = errorsx.SafeErrWrapperBuilder{
		Classifier: errorsx.ClassifyTLSFailure,
		Error:      err,
		Operation:  errorsx.TLSHandshakeOperation,
	}.MaybeBuild()
	return tlsconn, state, err
}

// EmitterTLSHandshaker emits events using the MeasurementRoot
type EmitterTLSHandshaker struct {
	TLSHandshaker
}

// Handshake implements Handshaker.Handshake
func (h EmitterTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	root := modelx.ContextMeasurementRootOrDefault(ctx)
	root.Handler.OnMeasurement(modelx.Measurement{
		TLSHandshakeStart: &modelx.TLSHandshakeStartEvent{
			DurationSinceBeginning: time.Since(root.Beginning),
			SNI:                    config.ServerName,
		},
	})
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	root.Handler.OnMeasurement(modelx.Measurement{
		TLSHandshakeDone: &modelx.TLSHandshakeDoneEvent{
			ConnectionState:        modelx.NewTLSConnectionState(state),
			Error:                  err,
			DurationSinceBeginning: time.Since(root.Beginning),
		},
	})
	return tlsconn, state, err
}
