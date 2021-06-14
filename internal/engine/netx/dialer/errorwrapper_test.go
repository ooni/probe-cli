package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/mockablex"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
)

func TestErrorWrapperFailure(t *testing.T) {
	ctx := context.Background()
	d := &errorWrapperDialer{Dialer: mockablex.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return nil, io.EOF
		},
	}}
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
		dialErr      *ErrDial
		readErr      *ErrRead
		writeErr     *ErrWrite
		closeErr     *ErrClose
		handshakeErr *tlsdialer.ErrTLSHandshake
	)
	switch op {
	case errorx.ConnectOperation:
		if !errors.As(err, &dialErr) {
			t.Fatal("unexpected wrapper")
		}
	case errorx.ReadOperation:
		if !errors.As(err, &readErr) {
			t.Fatal("unexpected wrapper")
		}
	case errorx.WriteOperation:
		if !errors.As(err, &writeErr) {
			t.Fatal("unexpected wrapper")
		}
	case errorx.CloseOperation:
		if !errors.As(err, &closeErr) {
			t.Fatal("unexpected wrapper")
		}
	case errorx.TLSHandshakeOperation:
		if !errors.As(err, &handshakeErr) {
			t.Fatal("unexpected wrapper")
		}
	default:
		t.Fatal("unexpected wrapper")
	}
}

func TestErrorWrapperSuccess(t *testing.T) {
	ctx := context.Background()
	d := &errorWrapperDialer{Dialer: mockablex.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return &mockablex.Conn{
				MockRead: func(b []byte) (int, error) {
					return 0, io.EOF
				},
				MockWrite: func(b []byte) (int, error) {
					return 0, io.EOF
				},
				MockClose: func() error {
					return io.EOF
				},
				MockLocalAddr: func() net.Addr {
					return &net.TCPAddr{Port: 12345}
				},
			}, nil
		},
	}}
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
