package dialer_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/dialid"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

func TestErrorWrapperFailure(t *testing.T) {
	ctx := dialid.WithDialID(context.Background())
	d := dialer.ErrorWrapperDialer{Dialer: dialer.EOFDialer{}}
	conn, err := d.DialContext(ctx, "tcp", "www.google.com:443")
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
	errorWrapperCheckErr(t, err, errorx.ConnectOperation)
}

func errorWrapperCheckErr(t *testing.T, err error, op string) {
	if !errors.As(err, &io.EOF) {
		t.Fatal("expected another error here")
	}
	var (
		dialErr      *dialer.ErrDial
		readErr      *dialer.ErrRead
		writeErr     *dialer.ErrWrite
		closeErr     *dialer.ErrClose
		handshakeErr *dialer.ErrTLSHandshake
	)
	switch op {
	case "dial":
		if !errors.As(err, &dialErr) {
			t.Fatal("unexpected wrapper")
		}
	case "read":
		if !errors.As(err, &readErr) {
			t.Fatal("unexpected wrapper")
		}
	case "write":
		if !errors.As(err, &writeErr) {
			t.Fatal("unexpected wrapper")
		}
	case "close":
		if !errors.As(err, &closeErr) {
			t.Fatal("unexpected wrapper")
		}
	case "tls_handshake":
		if !errors.As(err, &handshakeErr) {
			t.Fatal("unexpected wrapper")
		}
	}
}

func TestErrorWrapperSuccess(t *testing.T) {
	ctx := dialid.WithDialID(context.Background())
	d := dialer.ErrorWrapperDialer{Dialer: dialer.EOFConnDialer{}}
	conn, err := d.DialContext(ctx, "tcp", "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	count, err := conn.Read(nil)
	errorWrapperCheckIOResult(t, count, err, errorx.ReadOperation)
	count, err = conn.Write(nil)
	errorWrapperCheckIOResult(t, count, err, errorx.WriteOperation)
	err = conn.Close()
	errorWrapperCheckErr(t, err, errorx.CloseOperation)
}

func errorWrapperCheckIOResult(t *testing.T, count int, err error, op string) {
	if count != 0 {
		t.Fatal("expected nil count here")
	}
	errorWrapperCheckErr(t, err, op)
}
