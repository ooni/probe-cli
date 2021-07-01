package errorsx

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxmocks"
)

func TestErrorWrapperQUICDialerFailure(t *testing.T) {
	ctx := context.Background()
	d := &ErrorWrapperQUICDialer{Dialer: &netxmocks.QUICContextDialer{
		MockDialContext: func(ctx context.Context, network, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
			return nil, io.EOF
		},
	}}
	sess, err := d.DialContext(
		ctx, "udp", "www.google.com:443", &tls.Config{}, &quic.Config{})
	if sess != nil {
		t.Fatal("expected a nil sess here")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected another error here")
	}
	var errWrapper *ErrWrapper
	if !errors.As(err, &errWrapper) {
		t.Fatal("cannot cast to ErrWrapper")
	}
	if errWrapper.Operation != QUICHandshakeOperation {
		t.Fatal("unexpected Operation")
	}
	if errWrapper.Failure != FailureEOFError {
		t.Fatal("unexpected failure")
	}
}
