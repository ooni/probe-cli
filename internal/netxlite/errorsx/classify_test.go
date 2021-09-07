package errorsx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucas-clemente/quic-go"
	"github.com/pion/stun"
)

func TestClassifyGenericError(t *testing.T) {
	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if ClassifyGenericError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})
	t.Run("for already wrapped error", func(t *testing.T) {
		err := io.EOF
		if ClassifyGenericError(err) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for context.Canceled", func(t *testing.T) {
		if ClassifyGenericError(context.Canceled) != FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for operation was canceled error", func(t *testing.T) {
		if ClassifyGenericError(errors.New("operation was canceled")) != FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for EOF", func(t *testing.T) {
		if ClassifyGenericError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for canceled", func(t *testing.T) {
		if ClassifyGenericError(syscall.ECANCELED) != FailureOperationCanceled {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_refused", func(t *testing.T) {
		if ClassifyGenericError(syscall.ECONNREFUSED) != FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_reset", func(t *testing.T) {
		if ClassifyGenericError(syscall.ECONNRESET) != FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for host_unreachable", func(t *testing.T) {
		if ClassifyGenericError(syscall.EHOSTUNREACH) != FailureHostUnreachable {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for system timeout", func(t *testing.T) {
		if ClassifyGenericError(syscall.ETIMEDOUT) != FailureTimedOut {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for context deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel()
		<-ctx.Done()
		if ClassifyGenericError(ctx.Err()) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for stun's transaction is timed out", func(t *testing.T) {
		if ClassifyGenericError(stun.ErrTransactionTimeOut) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for i/o error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel() // fail immediately
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", "www.google.com:80")
		if err == nil {
			t.Fatal("expected an error here")
		}
		if conn != nil {
			t.Fatal("expected nil connection here")
		}
		if ClassifyGenericError(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for TLS handshake timeout error", func(t *testing.T) {
		err := errors.New("net/http: TLS handshake timeout")
		if ClassifyGenericError(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for no such host", func(t *testing.T) {
		if ClassifyGenericError(&net.DNSError{
			Err: "no such host",
		}) != FailureDNSNXDOMAINError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for errors including IPv4 address", func(t *testing.T) {
		input := errors.New("read tcp 10.0.2.15:56948->93.184.216.34:443: use of closed network connection")
		expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: use of closed network connection"
		out := ClassifyGenericError(input)
		if out != expected {
			t.Fatal(cmp.Diff(expected, out))
		}
	})
	t.Run("for errors including IPv6 address", func(t *testing.T) {
		input := errors.New("read tcp [::1]:56948->[::1]:443: use of closed network connection")
		expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: use of closed network connection"
		out := ClassifyGenericError(input)
		if out != expected {
			t.Fatal(cmp.Diff(expected, out))
		}
	})
	t.Run("for i/o error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel() // fail immediately
		udpAddr := &net.UDPAddr{IP: net.ParseIP("216.58.212.164"), Port: 80, Zone: ""}
		udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			t.Fatal(err)
		}
		sess, err := quic.DialEarlyContext(ctx, udpConn, udpAddr, "google.com:80", &tls.Config{}, &quic.Config{})
		if err == nil {
			t.Fatal("expected an error here")
		}
		if sess != nil {
			t.Fatal("expected nil session here")
		}
		if ClassifyGenericError(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
}

func TestClassifyQUICHandshakeError(t *testing.T) {
	t.Run("for input being already an ErrWrapper", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureEOFError}
		if ClassifyQUICHandshakeError(err) != FailureEOFError {
			t.Fatal("did not classify existing ErrWrapper correctly")
		}
	})
	t.Run("for connection_reset", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.StatelessResetError{}) != FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for incompatible quic version", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.VersionNegotiationError{}) != FailureQUICIncompatibleVersion {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for quic connection refused", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: quic.ConnectionRefused}) != FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for quic handshake timeout", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.HandshakeTimeoutError{}) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC idle connection timeout", func(t *testing.T) {
		if ClassifyQUICHandshakeError(&quic.IdleTimeoutError{}) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC CRYPTO Handshake", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertHandshakeFailure
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLFailedHandshake {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC CRYPTO Invalid Certificate", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertBadCertificate
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC CRYPTO Unknown CA", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSAlertUnknownCA
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLUnknownAuthority {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC CRYPTO Bad Hostname", func(t *testing.T) {
		var err quic.TransportErrorCode = quicTLSUnrecognizedName
		if ClassifyQUICHandshakeError(&quic.TransportError{ErrorCode: err}) != FailureSSLInvalidHostname {
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
	t.Run("for another kind of error", func(t *testing.T) {
		if ClassifyResolverError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
}

func TestClassifyTLSHandshakeError(t *testing.T) {
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
	t.Run("for another kind of error", func(t *testing.T) {
		if ClassifyTLSHandshakeError(io.EOF) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
}
