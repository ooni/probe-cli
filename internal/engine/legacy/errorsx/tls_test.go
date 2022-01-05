package errorsx

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestErrorWrapperTLSHandshakerFailure(t *testing.T) {
	th := ErrorWrapperTLSHandshaker{TLSHandshaker: &mocks.TLSHandshaker{
		MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
			return nil, tls.ConnectionState{}, io.EOF
		},
	}}
	conn, _, err := th.Handshake(
		context.Background(), &mocks.Conn{
			MockRead: func(b []byte) (int, error) {
				return 0, io.EOF
			},
		}, new(tls.Config))
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
	var errWrapper *netxlite.ErrWrapper
	if !errors.As(err, &errWrapper) {
		t.Fatal("cannot cast to ErrWrapper")
	}
	if errWrapper.Failure != netxlite.FailureEOFError {
		t.Fatal("unexpected Failure")
	}
	if errWrapper.Operation != netxlite.TLSHandshakeOperation {
		t.Fatal("unexpected Operation")
	}
}
