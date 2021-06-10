package errorx

import (
	"testing"
)

func TestToOperationString(t *testing.T) {
	t.Run("for connect", func(t *testing.T) {
		// You're doing HTTP and connect fails. You want to know
		// that connect failed not that HTTP failed.
		err := &ErrWrapper{Operation: ConnectOperation}
		if toOperationString(err, HTTPRoundTripOperation) != ConnectOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for http_round_trip", func(t *testing.T) {
		// You're doing DoH and something fails inside HTTP. You want
		// to know about the internal HTTP error, not resolve.
		err := &ErrWrapper{Operation: HTTPRoundTripOperation}
		if toOperationString(err, ResolveOperation) != HTTPRoundTripOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for resolve", func(t *testing.T) {
		// You're doing HTTP and the DNS fails. You want to
		// know that resolve failed.
		err := &ErrWrapper{Operation: ResolveOperation}
		if toOperationString(err, HTTPRoundTripOperation) != ResolveOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for tls_handshake", func(t *testing.T) {
		// You're doing HTTP and the TLS handshake fails. You want
		// to know about a TLS handshake error.
		err := &ErrWrapper{Operation: TLSHandshakeOperation}
		if toOperationString(err, HTTPRoundTripOperation) != TLSHandshakeOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for minor operation", func(t *testing.T) {
		// You just noticed that TLS handshake failed and you
		// have a child error telling you that read failed. Here
		// you want to know about a TLS handshake error.
		err := &ErrWrapper{Operation: ReadOperation}
		if toOperationString(err, TLSHandshakeOperation) != TLSHandshakeOperation {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for quic_handshake", func(t *testing.T) {
		// You're doing HTTP and the TLS handshake fails. You want
		// to know about a TLS handshake error.
		err := &ErrWrapper{Operation: QUICHandshakeOperation}
		if toOperationString(err, HTTPRoundTripOperation) != QUICHandshakeOperation {
			t.Fatal("unexpected result")
		}
	})
}
