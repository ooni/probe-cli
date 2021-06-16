package quicdialer_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/dialid"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/quicdialer"
)

func TestErrorWrapperFailure(t *testing.T) {
	ctx := dialid.WithDialID(context.Background())
	d := quicdialer.ErrorWrapperDialer{
		Dialer: MockDialer{Sess: nil, Err: io.EOF}}
	sess, err := d.DialContext(
		ctx, "udp", "www.google.com:443", &tls.Config{}, &quic.Config{})
	if sess != nil {
		t.Fatal("expected a nil sess here")
	}
	errorWrapperCheckErr(t, err, errorx.QUICHandshakeOperation)
}

func errorWrapperCheckErr(t *testing.T, err error, op string) {
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected another error here")
	}
	var errWrapper *errorx.ErrWrapper
	if !errors.As(err, &errWrapper) {
		t.Fatal("cannot cast to ErrWrapper")
	}
	if errWrapper.DialID == 0 {
		t.Fatal("unexpected DialID")
	}
	if errWrapper.Operation != op {
		t.Fatal("unexpected Operation")
	}
	if errWrapper.Failure != errorx.FailureEOFError {
		t.Fatal("unexpected failure")
	}
}
func TestErrorWrapperInvalidCertificate(t *testing.T) {
	nextprotos := []string{"h3"}
	servername := "example.com"
	tlsConf := &tls.Config{
		NextProtos: nextprotos,
		ServerName: servername,
	}

	dlr := quicdialer.ErrorWrapperDialer{Dialer: &quicdialer.SystemDialer{}}
	// use Google IP
	sess, err := dlr.DialContext(context.Background(), "udp",
		"216.58.212.164:443", tlsConf, &quic.Config{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if sess != nil {
		t.Fatal("expected nil sess here")
	}
	if err.Error() != errorx.FailureSSLInvalidCertificate {
		t.Fatal("unexpected failure")
	}
}

func TestErrorWrapperSuccess(t *testing.T) {
	ctx := dialid.WithDialID(context.Background())
	tlsConf := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "www.google.com",
	}
	d := quicdialer.ErrorWrapperDialer{Dialer: quicdialer.SystemDialer{}}
	sess, err := d.DialContext(ctx, "udp", "216.58.212.164:443", tlsConf, &quic.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sess == nil {
		t.Fatal("expected non-nil sess here")
	}
}

func TestClassifyQUICFailure(t *testing.T) {
	t.Run("for connection_reset", func(t *testing.T) {
		if quicdialer.ClassifyQUICFailure(&quic.StatelessResetError{}) != errorx.FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for incompatible quic version", func(t *testing.T) {
		if quicdialer.ClassifyQUICFailure(&quic.VersionNegotiationError{}) != errorx.FailureNoCompatibleQUICVersion {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for quic connection refused", func(t *testing.T) {
		if quicdialer.ClassifyQUICFailure(&quic.TransportError{ErrorCode: quic.ConnectionRefused}) != errorx.FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for quic handshake timeout", func(t *testing.T) {
		if quicdialer.ClassifyQUICFailure(&quic.HandshakeTimeoutError{}) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC idle connection timeout", func(t *testing.T) {
		if quicdialer.ClassifyQUICFailure(&quic.IdleTimeoutError{}) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC CRYPTO Invalid Certificate", func(t *testing.T) {
		var err quic.TransportErrorCode = 42
		if quicdialer.ClassifyQUICFailure(&quic.TransportError{ErrorCode: err}) != errorx.FailureSSLInvalidCertificate {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC CRYPTO Unknown CA", func(t *testing.T) {
		var err quic.TransportErrorCode = 48
		if quicdialer.ClassifyQUICFailure(&quic.TransportError{ErrorCode: err}) != errorx.FailureSSLUnknownAuthority {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC CRYPTO Bad Hostname", func(t *testing.T) {
		var err quic.TransportErrorCode = 112
		if quicdialer.ClassifyQUICFailure(&quic.TransportError{ErrorCode: err}) != errorx.FailureSSLInvalidHostname {
			t.Fatal("unexpected results")
		}
	})

}
