package errorsx

import (
	"context"
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
)

// QUICContextDialer is a dialer for QUIC using Context.
type QUICContextDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	DialContext(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)
}

// ErrorWrapperQUICDialer is a dialer that performs quic err wrapping
type ErrorWrapperQUICDialer struct {
	Dialer QUICContextDialer
}

// DialContext implements ContextDialer.DialContext
func (d *ErrorWrapperQUICDialer) DialContext(
	ctx context.Context, network string, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	sess, err := d.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	err = SafeErrWrapperBuilder{
		Classifier: ClassifyQUICFailure,
		Error:      err,
		Operation:  QUICHandshakeOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
	}
	return sess, nil
}
