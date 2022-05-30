package bytecounter

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestContextAwareDialer(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		dialAndUseConn := func(ctx context.Context, bufsiz int) error {
			childConn := &mocks.Conn{
				MockRead: func(b []byte) (int, error) {
					return len(b), nil
				},
				MockWrite: func(b []byte) (int, error) {
					return len(b), nil
				},
			}
			child := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return childConn, nil
				},
			}
			dialer := NewContextAwareDialer(child)
			conn, err := dialer.DialContext(ctx, "tcp", "10.0.0.1:443")
			if err != nil {
				return err
			}
			buffer := make([]byte, bufsiz)
			conn.Read(buffer)
			conn.Write(buffer)
			return nil
		}

		t.Run("normal usage", func(t *testing.T) {
			if testing.Short() {
				t.Skip("skip test in short mode")
			}
			sess := New()
			ctx := context.Background()
			ctx = WithSessionByteCounter(ctx, sess)
			const count = 128
			if err := dialAndUseConn(ctx, count); err != nil {
				t.Fatal(err)
			}
			exp := New()
			ctx = WithExperimentByteCounter(ctx, exp)
			if err := dialAndUseConn(ctx, count); err != nil {
				t.Fatal(err)
			}
			if exp.Received.Load() != count {
				t.Fatal("experiment should have received 128 bytes")
			}
			if sess.Received.Load() != 2*count {
				t.Fatal("session should have received 256 bytes")
			}
			if exp.Sent.Load() != count {
				t.Fatal("experiment should have sent 128 bytes")
			}
			if sess.Sent.Load() != 256 {
				t.Fatal("session should have sent 256 bytes")
			}
		})

		t.Run("failure", func(t *testing.T) {
			dialer := &ContextAwareDialer{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						return nil, io.EOF
					},
				},
			}
			conn, err := dialer.DialContext(context.Background(), "tcp", "www.google.com:80")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn here")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.Dialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		dialer := NewContextAwareDialer(child)
		dialer.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}
