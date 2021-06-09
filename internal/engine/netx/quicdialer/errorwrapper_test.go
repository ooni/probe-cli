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
	if !errors.As(err, &io.EOF) {
		t.Fatal("expected another error here")
	}
	var (
		dialErr     *quicdialer.ErrDial
		readfromErr *quicdialer.ErrReadFrom
		writetoErr  *quicdialer.ErrWriteTo
	)
	switch op {
	case "dial":
		if !errors.As(err, &dialErr) {
			t.Fatal("unexpected wrapper")
		}
	case "read_from":
		if !errors.As(err, &readfromErr) {
			t.Fatal("unexpected wrapper")
		}
	case "write_to":
		if !errors.As(err, &writetoErr) {
			t.Fatal("unexpected wrapper")
		}
	}
}

func TestErrorWrapperSuccess(t *testing.T) {
	ctx := dialid.WithDialID(context.Background())
	tlsConf := &tls.Config{
		NextProtos: []string{"h3-29"},
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
