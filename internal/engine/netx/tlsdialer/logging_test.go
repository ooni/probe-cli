package tlsdialer_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
)

func TestLoggingTLSHandshakerFailure(t *testing.T) {
	h := tlsdialer.LoggingTLSHandshaker{
		TLSHandshaker: tlsdialer.EOFTLSHandshaker{},
		Logger:        log.Log,
	}
	tlsconn, _, err := h.Handshake(context.Background(), tlsdialer.EOFConn{}, &tls.Config{
		ServerName: "www.google.com",
	})
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if tlsconn != nil {
		t.Fatal("expected nil tlsconn here")
	}
}
