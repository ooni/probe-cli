package errorsx

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestErrorWrapperDialerFailure(t *testing.T) {
	ctx := context.Background()
	d := &ErrorWrapperDialer{Dialer: &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return nil, io.EOF
		},
	}}
	conn, err := d.DialContext(ctx, "tcp", "www.google.com:443")
	var ew *netxlite.ErrWrapper
	if !errors.As(err, &ew) {
		t.Fatal("cannot convert to ErrWrapper")
	}
	if ew.Operation != netxlite.ConnectOperation {
		t.Fatal("unexpected operation", ew.Operation)
	}
	if ew.Failure != netxlite.FailureEOFError {
		t.Fatal("unexpected failure", ew.Failure)
	}
	if !errors.Is(ew.WrappedErr, io.EOF) {
		t.Fatal("unexpected underlying error", ew.WrappedErr)
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestErrorWrapperDialerSuccess(t *testing.T) {
	origConn := &net.TCPConn{}
	ctx := context.Background()
	d := &ErrorWrapperDialer{Dialer: &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return origConn, nil
		},
	}}
	conn, err := d.DialContext(ctx, "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	ewConn, ok := conn.(*errorWrapperConn)
	if !ok {
		t.Fatal("cannot cast to errorWrapperConn")
	}
	if ewConn.Conn != origConn {
		t.Fatal("not the connection we expected")
	}
}

func TestErrorWrapperConnReadFailure(t *testing.T) {
	c := &errorWrapperConn{
		Conn: &mocks.Conn{
			MockRead: func(b []byte) (int, error) {
				return 0, io.EOF
			},
		},
	}
	buf := make([]byte, 1024)
	cnt, err := c.Read(buf)
	var ew *netxlite.ErrWrapper
	if !errors.As(err, &ew) {
		t.Fatal("cannot cast error to ErrWrapper")
	}
	if ew.Operation != netxlite.ReadOperation {
		t.Fatal("invalid operation", ew.Operation)
	}
	if ew.Failure != netxlite.FailureEOFError {
		t.Fatal("invalid failure", ew.Failure)
	}
	if !errors.Is(ew.WrappedErr, io.EOF) {
		t.Fatal("invalid wrapped error", ew.WrappedErr)
	}
	if cnt != 0 {
		t.Fatal("expected zero here", cnt)
	}
}

func TestErrorWrapperConnReadSuccess(t *testing.T) {
	c := &errorWrapperConn{
		Conn: &mocks.Conn{
			MockRead: func(b []byte) (int, error) {
				return len(b), nil
			},
		},
	}
	buf := make([]byte, 1024)
	cnt, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if cnt != len(buf) {
		t.Fatal("expected len(buf) here", cnt)
	}
}

func TestErrorWrapperConnWriteFailure(t *testing.T) {
	c := &errorWrapperConn{
		Conn: &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return 0, io.EOF
			},
		},
	}
	buf := make([]byte, 1024)
	cnt, err := c.Write(buf)
	var ew *netxlite.ErrWrapper
	if !errors.As(err, &ew) {
		t.Fatal("cannot cast error to ErrWrapper")
	}
	if ew.Operation != netxlite.WriteOperation {
		t.Fatal("invalid operation", ew.Operation)
	}
	if ew.Failure != netxlite.FailureEOFError {
		t.Fatal("invalid failure", ew.Failure)
	}
	if !errors.Is(ew.WrappedErr, io.EOF) {
		t.Fatal("invalid wrapped error", ew.WrappedErr)
	}
	if cnt != 0 {
		t.Fatal("expected zero here", cnt)
	}
}

func TestErrorWrapperConnWriteSuccess(t *testing.T) {
	c := &errorWrapperConn{
		Conn: &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return len(b), nil
			},
		},
	}
	buf := make([]byte, 1024)
	cnt, err := c.Write(buf)
	if err != nil {
		t.Fatal(err)
	}
	if cnt != len(buf) {
		t.Fatal("expected len(buf) here", cnt)
	}
}

func TestErrorWrapperConnCloseFailure(t *testing.T) {
	c := &errorWrapperConn{
		Conn: &mocks.Conn{
			MockClose: func() error {
				return io.EOF
			},
		},
	}
	err := c.Close()
	var ew *netxlite.ErrWrapper
	if !errors.As(err, &ew) {
		t.Fatal("cannot cast error to ErrWrapper")
	}
	if ew.Operation != netxlite.CloseOperation {
		t.Fatal("invalid operation", ew.Operation)
	}
	if ew.Failure != netxlite.FailureEOFError {
		t.Fatal("invalid failure", ew.Failure)
	}
	if !errors.Is(ew.WrappedErr, io.EOF) {
		t.Fatal("invalid wrapped error", ew.WrappedErr)
	}
}

func TestErrorWrapperConnCloseSuccess(t *testing.T) {
	c := &errorWrapperConn{
		Conn: &mocks.Conn{
			MockClose: func() error {
				return nil
			},
		},
	}
	err := c.Close()
	if err != nil {
		t.Fatal(err)
	}
}
