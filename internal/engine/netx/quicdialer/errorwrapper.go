package quicdialer

import (
	"context"
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
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
	sess, err := d.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	err = errorx.SafeErrWrapperBuilder{
		Classifier: errorx.ClassifyQUICFailure,
		Error:      err,
		Operation:  errorx.QUICHandshakeOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
	}
	return sess, nil
}
