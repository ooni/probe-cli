package errorx

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

func TestMaybeBuildFactory(t *testing.T) {
	err := SafeErrWrapperBuilder{
		ConnID:        1,
		DialID:        10,
		Error:         errors.New("mocked error"),
		TransactionID: 100,
	}.MaybeBuild()
	var target *ErrWrapper
	if errors.As(err, &target) == false {
		t.Fatal("not the expected error type")
	}
	if target.ConnID != 1 {
		t.Fatal("wrong ConnID")
	}
	if target.DialID != 10 {
		t.Fatal("wrong DialID")
	}
	if target.Failure != "unknown_failure: mocked error" {
		t.Fatal("the failure string is wrong")
	}
	if target.TransactionID != 100 {
		t.Fatal("the transactionID is wrong")
	}
	if target.WrappedErr.Error() != "mocked error" {
		t.Fatal("the wrapped error is wrong")
	}
}

func TestToFailureString(t *testing.T) {
	t.Run("for already wrapped error", func(t *testing.T) {
		err := SafeErrWrapperBuilder{Error: io.EOF}.MaybeBuild()
		if toFailureString(err) != FailureEOFError {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for ErrDNSBogon", func(t *testing.T) {
		if toFailureString(ErrDNSBogon) != FailureDNSBogonError {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for context.Canceled", func(t *testing.T) {
		if toFailureString(context.Canceled) != FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for x509.HostnameError", func(t *testing.T) {
		var err x509.HostnameError
		if toFailureString(err) != FailureSSLInvalidHostname {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for x509.UnknownAuthorityError", func(t *testing.T) {
		var err x509.UnknownAuthorityError
		if toFailureString(err) != FailureSSLUnknownAuthority {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for x509.CertificateInvalidError", func(t *testing.T) {
		var err x509.CertificateInvalidError
		if toFailureString(err) != FailureSSLInvalidCertificate {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for operation was canceled error", func(t *testing.T) {
		if toFailureString(errors.New("operation was canceled")) != FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for EOF", func(t *testing.T) {
		if toFailureString(io.EOF) != FailureEOFError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_refused", func(t *testing.T) {
		if toFailureString(syscall.ECONNREFUSED) != FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_reset", func(t *testing.T) {
		if toFailureString(syscall.ECONNRESET) != FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for context deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel()
		<-ctx.Done()
		if toFailureString(ctx.Err()) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for stun's transaction is timed out", func(t *testing.T) {
		if toFailureString(stun.ErrTransactionTimeOut) != FailureGenericTimeoutError {
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
		if toFailureString(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for TLS handshake timeout error", func(t *testing.T) {
		err := errors.New("net/http: TLS handshake timeout")
		if toFailureString(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for no such host", func(t *testing.T) {
		if toFailureString(&net.DNSError{
			Err: "no such host",
		}) != FailureDNSNXDOMAINError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for errors including IPv4 address", func(t *testing.T) {
		input := errors.New("read tcp 10.0.2.15:56948->93.184.216.34:443: use of closed network connection")
		expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: use of closed network connection"
		out := toFailureString(input)
		if out != expected {
			t.Fatal(cmp.Diff(expected, out))
		}
	})
	t.Run("for errors including IPv6 address", func(t *testing.T) {
		input := errors.New("read tcp [::1]:56948->[::1]:443: use of closed network connection")
		expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: use of closed network connection"
		out := toFailureString(input)
		if out != expected {
			t.Fatal(cmp.Diff(expected, out))
		}
	})
	// QUIC failures
	t.Run("for connection_refused", func(t *testing.T) {
		if toFailureString(errors.New("connection_refused")) != FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_reset", func(t *testing.T) {
		if toFailureString(errors.New("stateless_reset")) != FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for incompatible quic version", func(t *testing.T) {
		if toFailureString(errors.New("No compatible QUIC version found")) != FailureNoCompatibleQUICVersion {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for i/o error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel() // fail immediately
		udpAddr := &net.UDPAddr{IP: net.ParseIP("216.58.212.164"), Port: 80, Zone: ""}
		udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		sess, err := quic.DialEarlyContext(ctx, udpConn, udpAddr, "google.com:80", &tls.Config{}, &quic.Config{})
		if err == nil {
			t.Fatal("expected an error here")
		}
		if sess != nil {
			t.Fatal("expected nil session here")
		}
		if toFailureString(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC handshake timeout error", func(t *testing.T) {
		err := errors.New("Handshake did not complete in time")
		if toFailureString(err) != FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
}

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
