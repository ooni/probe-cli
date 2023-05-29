package netxlite

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pion/stun"
	"github.com/quic-go/quic-go"
)

func TestClassifyGenericError(t *testing.T) {
	// Please, keep this list sorted in the same order
	// in which checks appear on the code

	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if ClassifyGenericError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})

	t.Run("for a system call error", func(t *testing.T) {
		if ClassifyGenericError(EWOULDBLOCK) != FailureOperationWouldBlock {
			t.Fatal("unexpected results")
		}
	})

	// Now we enter into classifyWithStringSuffix. We test it here
	// since we want to test the ClassifyGenericError in is
	// entirety here and the classifyWithStringSuffix function
	// is just an implementation detail.

	t.Run("for operation was canceled", func(t *testing.T) {
		if ClassifyGenericError(errors.New("operation was canceled")) != FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for EOF", func(t *testing.T) {
		if ClassifyGenericError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for context deadline exceeded", func(t *testing.T) {
		if ClassifyGenericError(context.DeadlineExceeded) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for stun's transaction is timed out", func(t *testing.T) {
		if ClassifyGenericError(stun.ErrTransactionTimeOut) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for i/o timeout", func(t *testing.T) {
		if ClassifyGenericError(errors.New("i/o timeout")) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for TLS handshake timeout", func(t *testing.T) {
		err := errors.New("net/http: TLS handshake timeout")
		if ClassifyGenericError(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for no such host", func(t *testing.T) {
		if ClassifyGenericError(errors.New("no such host")) != FailureDNSNXDOMAINError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for dns server misbehaving", func(t *testing.T) {
		if ClassifyGenericError(errors.New("dns server misbehaving")) != FailureDNSServerMisbehaving {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for no answer from DNS server", func(t *testing.T) {
		if ClassifyGenericError(errors.New("no answer from DNS server")) != FailureDNSNoAnswer {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for use of closed network connection", func(t *testing.T) {
		err := errors.New("read tcp 10.0.2.15:56948->93.184.216.34:443: use of closed network connection")
		if ClassifyGenericError(err) != FailureConnectionAlreadyClosed {
			t.Fatal("unexpected results")
		}
	})

	// Now we're back in ClassifyGenericError

	t.Run("for context.Canceled", func(t *testing.T) {
		if ClassifyGenericError(context.Canceled) != FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for the 'unknown_failure' string", func(t *testing.T) {
		if ClassifyGenericError(ErrUnknown) != FailureUnknown {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for unknown errors", func(t *testing.T) {
		t.Run("with an IPv4 address", func(t *testing.T) {
			input := errors.New("read tcp 10.0.2.15:56948->93.184.216.34:443: some error")
			expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: some error"
			out := ClassifyGenericError(input)
			if out != expected {
				t.Fatal(cmp.Diff(expected, out))
			}
		})

		t.Run("with an IPv6 address", func(t *testing.T) {
			input := errors.New("read tcp [::1]:56948->[::1]:443: some error")
			expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: some error"
			out := ClassifyGenericError(input)
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
		if ClassifyQUICHandshakeError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})

	t.Run("for incompatible quic version", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.VersionNegotiationError{}) != FailureQUICIncompatibleVersion {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for stateless reset", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.StatelessResetError{}) != FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for handshake timeout", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.HandshakeTimeoutError{}) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for idle timeout", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.IdleTimeoutError{}) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for connection refused", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: quic.ConnectionRefused}) != FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for bad certificate", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertBadCertificate
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for unsupported certificate", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertUnsupportedCertificate
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for certificate expired", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertCertificateExpired
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for certificate revoked", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertCertificateRevoked
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for certificate unknown", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertCertificateUnknown
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for decrypt error", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertDecryptError
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLFailedHandshake {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for handshake failure", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertHandshakeFailure
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLFailedHandshake {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for unknown CA", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertUnknownCA
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLUnknownAuthority {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for unrecognized hostname", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSUnrecognizedName
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidHostname {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for a TransportError wrapping an OONI error", func(t *testing.T) {
		err := &quic.TransportError{
			ErrorCode:    quic.InternalError,
			ErrorMessage: FailureHostUnreachable,
		}
		if ClassifyQUICHandshakeError(err) != FailureHostUnreachable {
			t.Fatal("unexpected results")
		}
	})

	t.Run("for another kind of error", func(t *testing.T) {
		if ClassifyQUICHandshakeError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
}

func TestClassifyResolverError(t *testing.T) {
	// Please, keep this list sorted in the same order
	// in which checks appear on the code

	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if ClassifyResolverError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})

	t.Run("for ErrDNSBogon", func(t *testing.T) {
		if ClassifyResolverError(ErrDNSBogon) != FailureDNSBogonError {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for refused", func(t *testing.T) {
		if ClassifyResolverError(ErrOODNSRefused) != FailureDNSRefusedError {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for servfail", func(t *testing.T) {
		if ClassifyResolverError(ErrOODNSServfail) != FailureDNSServfailError {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for dns reply with wrong queryID", func(t *testing.T) {
		if ClassifyResolverError(ErrDNSReplyWithWrongQueryID) != FailureDNSReplyWithWrongQueryID {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for EAI_NODATA returned by Android's getaddrinfo", func(t *testing.T) {
		if ClassifyResolverError(ErrAndroidDNSCacheNoData) != FailureAndroidDNSCacheNoData {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for another kind of error", func(t *testing.T) {
		if ClassifyResolverError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
}

func TestClassifyTLSHandshakeError(t *testing.T) {
	// Please, keep this list sorted in the same order
	// in which checks appear on the code

	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if ClassifyTLSHandshakeError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})

	t.Run("for x509.HostnameError", func(t *testing.T) {
		var err x509.HostnameError
		if ClassifyTLSHandshakeError(err) != FailureSSLInvalidHostname {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for x509.UnknownAuthorityError", func(t *testing.T) {
		var err x509.UnknownAuthorityError
		if ClassifyTLSHandshakeError(err) != FailureSSLUnknownAuthority {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for x509.CertificateInvalidError", func(t *testing.T) {
		var err x509.CertificateInvalidError
		if ClassifyTLSHandshakeError(err) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for 'tls: unrecognized name' error", func(t *testing.T) {
		err := errors.New("tls: handshake failed: tls: unrecognized name")
		if ClassifyTLSHandshakeError(err) != FailureSSLInvalidHostname {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for 'tls: alert(112)' error", func(t *testing.T) { // yawning utls
		err := errors.New("tls: handshake failed: tls: alert(112)")
		if ClassifyTLSHandshakeError(err) != FailureSSLInvalidHostname {
			t.Fatal("unexpected result")
		}
	})

	t.Run("for another kind of error", func(t *testing.T) {
		if ClassifyTLSHandshakeError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
}
