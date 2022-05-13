package netxlite

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucas-clemente/quic-go"
	"github.com/pion/stun"
)

func TestClassifyGenericError(t *testing.T) {
	// Please, keep this list sorted in the same order
	// in which checks appear on the code

	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if classifyGenericError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})

	t.Run("for a system call error", func(t *testing.T) {
		if classifyGenericError(EWOULDBLOCK) != FailureOperationWouldBlock {
			t.Fatal("unexpected results")
		}
	})

	// Now we enter into classifyWithStringSuffix. We test it here
	// since we want to test the ClassifyGenericError in is
	// entirety here and the classifyWithStringSuffix function
	// is just an implementation detail.

	t.Run("for operation was canceled", func(t *testing.T) {
		if classifyGenericError(errors.New("operation was canceled")) != FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for EOF", func(t *testing.T) {
		if classifyGenericError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for context deadline exceeded", func(t *testing.T) {
		if classifyGenericError(context.DeadlineExceeded) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for stun's transaction is timed out", func(t *testing.T) {
		if classifyGenericError(stun.ErrTransactionTimeOut) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for i/o timeout", func(t *testing.T) {
		if classifyGenericError(errors.New("i/o timeout")) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for TLS handshake timeout", func(t *testing.T) {
		err := errors.New("net/http: TLS handshake timeout")
		if classifyGenericError(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for no such host", func(t *testing.T) {
		if classifyGenericError(errors.New("no such host")) != FailureDNSNXDOMAINError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for dns server misbehaving", func(t *testing.T) {
		if classifyGenericError(errors.New("dns server misbehaving")) != FailureDNSServerMisbehaving {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for no answer from DNS server", func(t *testing.T) {
		if classifyGenericError(errors.New("no answer from DNS server")) != FailureDNSNoAnswer {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for use of closed network connection", func(t *testing.T) {
		err := errors.New("read tcp 10.0.2.15:56948->93.184.216.34:443: use of closed network connection")
		if classifyGenericError(err) != FailureConnectionAlreadyClosed {
			t.Fatal("unexpected results")
		}
	})

	// Now we're back in ClassifyGenericError

	t.Run("for context.Canceled", func(t *testing.T) {
		if classifyGenericError(context.Canceled) != FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for unknown errors", func(t *testing.T) {
		t.Run("with an IPv4 address", func(t *testing.T) {
			input := errors.New("read tcp 10.0.2.15:56948->93.184.216.34:443: some error")
			expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: some error"
			out := classifyGenericError(input)
			if out != expected {
				t.Fatal(cmp.Diff(expected, out))
			}
		})

		t.Run("with an IPv6 address", func(t *testing.T) {
			input := errors.New("read tcp [::1]:56948->[::1]:443: some error")
			expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: some error"
			out := classifyGenericError(input)
			if out != expected {
				t.Fatal(cmp.Diff(expected, out))
			}
		})
	})
}

func TestClassifyQUICHandshakeError(t *testing.T) {
	// Please, keep this list sorted in the same order
	// in which checks appear on the code

	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if classifyQUICHandshakeError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})

	t.Run("for incompatible quic version", func(t *testing.T) {
		if classifyQUICHandshakeError(&quic.VersionNegotiationError{}) != FailureQUICIncompatibleVersion {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for stateless reset", func(t *testing.T) {
		if classifyQUICHandshakeError(&quic.StatelessResetError{}) != FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for handshake timeout", func(t *testing.T) {
		if classifyQUICHandshakeError(&quic.HandshakeTimeoutError{}) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for idle timeout", func(t *testing.T) {
		if classifyQUICHandshakeError(&quic.IdleTimeoutError{}) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for connection refused", func(t *testing.T) {
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: quic.ConnectionRefused}) != FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for bad certificate", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertBadCertificate
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for unsupported certificate", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertUnsupportedCertificate
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for certificate expired", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertCertificateExpired
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for certificate revoked", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertCertificateRevoked
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for certificate unknown", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertCertificateUnknown
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for decrypt error", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertDecryptError
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLFailedHandshake {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for handshake failure", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertHandshakeFailure
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLFailedHandshake {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for unknown CA", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertUnknownCA
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLUnknownAuthority {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for unrecognized hostname", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSUnrecognizedName
		if classifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidHostname {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for a TransportError wrapping an OONI error", func(t *testing.T) {
		err := &quic.TransportError{
			ErrorCode:    quic.InternalError,
			ErrorMessage: FailureHostUnreachable,
		}
		if classifyQUICHandshakeError(err) != FailureHostUnreachable {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for another kind of error", func(t *testing.T) {
		if classifyQUICHandshakeError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
}

func TestClassifyResolverError(t *testing.T) {
	// Please, keep this list sorted in the same order
	// in which checks appear on the code

	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if classifyResolverError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})

	t.Run("for ErrDNSBogon", func(t *testing.T) {
		if classifyResolverError(ErrDNSBogon) != FailureDNSBogonError {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for refused", func(t *testing.T) {
		if classifyResolverError(ErrOODNSRefused) != FailureDNSRefusedError {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for servfail", func(t *testing.T) {
		if classifyResolverError(ErrOODNSServfail) != FailureDNSServfailError {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for another kind of error", func(t *testing.T) {
		if classifyResolverError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
}

func TestClassifyTLSHandshakeError(t *testing.T) {
	// Please, keep this list sorted in the same order
	// in which checks appear on the code

	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if classifyTLSHandshakeError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})

	t.Run("for x509.HostnameError", func(t *testing.T) {
		var err x509.HostnameError
		if classifyTLSHandshakeError(err) != FailureSSLInvalidHostname {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for x509.UnknownAuthorityError", func(t *testing.T) {
		var err x509.UnknownAuthorityError
		if classifyTLSHandshakeError(err) != FailureSSLUnknownAuthority {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for x509.CertificateInvalidError", func(t *testing.T) {
		var err x509.CertificateInvalidError
		if classifyTLSHandshakeError(err) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for another kind of error", func(t *testing.T) {
		if classifyTLSHandshakeError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
}
