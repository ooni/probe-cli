// Package tlsdialer contains code to establish TLS connections.
package tlsdialer

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/connid"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
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

// SystemTLSHandshaker is the system TLS handshaker.
type SystemTLSHandshaker struct{}

// Handshake implements Handshaker.Handshake
func (h SystemTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	tlsconn := tls.Client(conn, config)
	if err := tlsconn.Handshake(); err != nil {
		return nil, tls.ConnectionState{}, err
	}
	return tlsconn, tlsconn.ConnectionState(), nil
}

// TimeoutTLSHandshaker is a TLSHandshaker with timeout
type TimeoutTLSHandshaker struct {
	TLSHandshaker
	HandshakeTimeout time.Duration // default: 10 second
}

// Handshake implements Handshaker.Handshake
func (h TimeoutTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	timeout := 10 * time.Second
	if h.HandshakeTimeout != 0 {
		timeout = h.HandshakeTimeout
	}
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, tls.ConnectionState{}, err
	}
	tlsconn, connstate, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	conn.SetDeadline(time.Time{})
	return tlsconn, connstate, err
}

// ErrorWrapperTLSHandshaker wraps the returned error to be an OONI error
type ErrorWrapperTLSHandshaker struct {
	TLSHandshaker
}

// Handshake implements Handshaker.Handshake
func (h ErrorWrapperTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	connID := connid.Compute(conn.RemoteAddr().Network(), conn.RemoteAddr().String())
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	err = errorx.SafeErrWrapperBuilder{
		Classifier: errorx.ClassifyTLSFailure,
		ConnID:     connID,
		Error:      err,
		Operation:  errorx.TLSHandshakeOperation,
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
	connID := connid.Compute(conn.RemoteAddr().Network(), conn.RemoteAddr().String())
	root := modelx.ContextMeasurementRootOrDefault(ctx)
	root.Handler.OnMeasurement(modelx.Measurement{
		TLSHandshakeStart: &modelx.TLSHandshakeStartEvent{
			ConnID:                 connID,
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
			SNI:                    config.ServerName,
		},
	})
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	root.Handler.OnMeasurement(modelx.Measurement{
		TLSHandshakeDone: &modelx.TLSHandshakeDoneEvent{
			ConnID:                 connID,
			ConnectionState:        modelx.NewTLSConnectionState(state),
			Error:                  err,
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
		},
	})
	return tlsconn, state, err
}

// TLSDialer is the TLS dialer
type TLSDialer struct {
	Config        *tls.Config
	Dialer        UnderlyingDialer
	TLSHandshaker TLSHandshaker
}

// DialTLSContext is like tls.DialTLS but with the signature of net.Dialer.DialContext
func (d TLSDialer) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	// Implementation note: when DialTLS is not set, the code in
	// net/http will perform the handshake. Otherwise, if DialTLS
	// is set, we will end up here. This code is still used when
	// performing non-HTTP TLS-enabled dial operations.
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	config := d.Config
	if config == nil {
		config = new(tls.Config)
	} else {
		config = config.Clone()
	}
	if config.ServerName == "" {
		config.ServerName = host
	}
	tlsconn, _, err := d.TLSHandshaker.Handshake(ctx, conn, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return tlsconn, nil
}
