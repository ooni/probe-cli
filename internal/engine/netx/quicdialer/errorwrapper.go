package quicdialer

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/dialid"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

// ErrorWrapperDialer is a dialer that performs quic err wrapping
type ErrorWrapperDialer struct {
	Dialer ContextDialer
}

// DialContext implements ContextDialer.DialContext
func (d ErrorWrapperDialer) DialContext(
	ctx context.Context, network string, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	dialID := dialid.ContextDialID(ctx)
	sess, err := d.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	err = errorx.SafeErrWrapperBuilder{
		// ConnID does not make any sense if we've failed and the error
		// does not make any sense (and is nil) if we succeeded.
		DialID:    dialID,
		Error:     err,
		Failure:   ClassifyQUICFailure(err),
		Operation: errorx.QUICHandshakeOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func ClassifyQUICFailure(err error) string {
	var versionNegotiation *quic.VersionNegotiationError
	var statelessReset *quic.StatelessResetError
	var handshakeTimeout *quic.HandshakeTimeoutError
	var idleTimeout *quic.IdleTimeoutError
	var transportError *quic.TransportError

	if errors.As(err, &versionNegotiation) {
		return errorx.FailureNoCompatibleQUICVersion
	}
	if errors.As(err, &statelessReset) {
		return errorx.FailureConnectionReset
	}
	if errors.As(err, &handshakeTimeout) {
		return errorx.FailureGenericTimeoutError
	}
	if errors.As(err, &idleTimeout) {
		return errorx.FailureGenericTimeoutError
	}
	if errors.As(err, &transportError) {
		if transportError.ErrorCode == quic.ConnectionRefused {
			return errorx.FailureConnectionRefused
		}
	}
	return ""
}
