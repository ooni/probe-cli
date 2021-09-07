package errorsx

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

func TestMaybeBuildFactory(t *testing.T) {
	err := SafeErrWrapperBuilder{
		Error: errors.New("mocked error"),
	}.MaybeBuild()
	var target *errorsx.ErrWrapper
	if errors.As(err, &target) == false {
		t.Fatal("not the expected error type")
	}
	if target.Failure != "unknown_failure: mocked error" {
		t.Fatal("the failure string is wrong")
	}
	if target.WrappedErr.Error() != "mocked error" {
		t.Fatal("the wrapped error is wrong")
	}
}

func TestToOperationString(t *testing.T) {
	t.Run("for connect", func(t *testing.T) {
		// You're doing HTTP and connect fails. You want to know
		// that connect failed not that HTTP failed.
		err := &errorsx.ErrWrapper{Operation: errorsx.ConnectOperation}
		if toOperationString(err, errorsx.HTTPRoundTripOperation) != errorsx.ConnectOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for http_round_trip", func(t *testing.T) {
		// You're doing DoH and something fails inside HTTP. You want
		// to know about the internal HTTP error, not resolve.
		err := &errorsx.ErrWrapper{Operation: errorsx.HTTPRoundTripOperation}
		if toOperationString(err, errorsx.ResolveOperation) != errorsx.HTTPRoundTripOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for resolve", func(t *testing.T) {
		// You're doing HTTP and the DNS fails. You want to
		// know that resolve failed.
		err := &errorsx.ErrWrapper{Operation: errorsx.ResolveOperation}
		if toOperationString(err, errorsx.HTTPRoundTripOperation) != errorsx.ResolveOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for tls_handshake", func(t *testing.T) {
		// You're doing HTTP and the TLS handshake fails. You want
		// to know about a TLS handshake error.
		err := &errorsx.ErrWrapper{Operation: errorsx.TLSHandshakeOperation}
		if toOperationString(err, errorsx.HTTPRoundTripOperation) != errorsx.TLSHandshakeOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for minor operation", func(t *testing.T) {
		// You just noticed that TLS handshake failed and you
		// have a child error telling you that read failed. Here
		// you want to know about a TLS handshake error.
		err := &errorsx.ErrWrapper{Operation: errorsx.ReadOperation}
		if toOperationString(err, errorsx.TLSHandshakeOperation) != errorsx.TLSHandshakeOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for quic_handshake", func(t *testing.T) {
		// You're doing HTTP and the TLS handshake fails. You want
		// to know about a TLS handshake error.
		err := &errorsx.ErrWrapper{Operation: errorsx.QUICHandshakeOperation}
		if toOperationString(err, errorsx.HTTPRoundTripOperation) != errorsx.QUICHandshakeOperation {
			t.Fatal("unexpected result")
		}
	})
}
