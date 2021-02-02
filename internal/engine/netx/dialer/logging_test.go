package dialer_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
)

func TestLoggingDialerFailure(t *testing.T) {
	d := dialer.LoggingDialer{
		Dialer: dialer.EOFDialer{},
		Logger: log.Log,
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestLoggingTLSHandshakerFailure(t *testing.T) {
	h := dialer.LoggingTLSHandshaker{
		TLSHandshaker: dialer.EOFTLSHandshaker{},
		Logger:        log.Log,
	}
	tlsconn, _, err := h.Handshake(context.Background(), dialer.EOFConn{}, &tls.Config{
		ServerName: "www.google.com",
	})
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if tlsconn != nil {
		t.Fatal("expected nil tlsconn here")
	}
}
