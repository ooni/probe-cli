package netxlite

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
)

func TestHTTPDialerWithReadTimeout(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			var (
				calledWithZeroTime    bool
				calledWithNonZeroTime bool
			)
			origConn := &mocks.Conn{
				MockSetReadDeadline: func(t time.Time) error {
					switch t.IsZero() {
					case true:
						calledWithZeroTime = true
					case false:
						calledWithNonZeroTime = true
					}
					return nil
				},
				MockRead: func(b []byte) (int, error) {
					return 0, io.EOF
				},
			}
			d := &httpDialerWithReadTimeout{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return origConn, nil
					},
				},
			}
			ctx := context.Background()
			conn, err := d.DialContext(ctx, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if _, okay := conn.(*httpConnWithReadTimeout); !okay {
				t.Fatal("invalid conn type")
			}
			if conn.(*httpConnWithReadTimeout).Conn != origConn {
				t.Fatal("invalid origin conn")
			}
			b := make([]byte, 1024)
			count, err := conn.Read(b)
			if !errors.Is(err, io.EOF) {
				t.Fatal("invalid error")
			}
			if count != 0 {
				t.Fatal("invalid count")
			}
			if !calledWithZeroTime || !calledWithNonZeroTime {
				t.Fatal("not called")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			d := &httpDialerWithReadTimeout{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return nil, expected
					},
				},
			}
			conn, err := d.DialContext(context.Background(), "", "")
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn here")
			}
		})
	})
}

func TestHTTPTLSDialerWithReadTimeout(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			var (
				calledWithZeroTime    bool
				calledWithNonZeroTime bool
			)
			origConn := &mocks.TLSConn{
				Conn: mocks.Conn{
					MockSetReadDeadline: func(t time.Time) error {
						switch t.IsZero() {
						case true:
							calledWithZeroTime = true
						case false:
							calledWithNonZeroTime = true
						}
						return nil
					},
					MockRead: func(b []byte) (int, error) {
						return 0, io.EOF
					},
				},
			}
			d := &httpTLSDialerWithReadTimeout{
				TLSDialer: &mocks.TLSDialer{
					MockDialTLSContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return origConn, nil
					},
				},
			}
			ctx := context.Background()
			conn, err := d.DialTLSContext(ctx, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if _, okay := conn.(*httpTLSConnWithReadTimeout); !okay {
				t.Fatal("invalid conn type")
			}
			if conn.(*httpTLSConnWithReadTimeout).TLSConn != origConn {
				t.Fatal("invalid origin conn")
			}
			b := make([]byte, 1024)
			count, err := conn.Read(b)
			if !errors.Is(err, io.EOF) {
				t.Fatal("invalid error")
			}
			if count != 0 {
				t.Fatal("invalid count")
			}
			if !calledWithZeroTime || !calledWithNonZeroTime {
				t.Fatal("not called")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			d := &httpTLSDialerWithReadTimeout{
				TLSDialer: &mocks.TLSDialer{
					MockDialTLSContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return nil, expected
					},
				},
			}
			conn, err := d.DialTLSContext(context.Background(), "", "")
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn here")
			}
		})

		t.Run("with invalid conn type", func(t *testing.T) {
			var called bool
			d := &httpTLSDialerWithReadTimeout{
				TLSDialer: &mocks.TLSDialer{
					MockDialTLSContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return &mocks.Conn{
							MockClose: func() error {
								called = true
								return nil
							},
						}, nil
					},
				},
			}
			conn, err := d.DialTLSContext(context.Background(), "", "")
			if !errors.Is(err, ErrNotTLSConn) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn here")
			}
			if !called {
				t.Fatal("not called")
			}
		})
	})
}
