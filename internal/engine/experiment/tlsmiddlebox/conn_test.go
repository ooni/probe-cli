package tlsmiddlebox

import (
	"context"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDialerTTLWrapperConn(t *testing.T) {
	t.Run("Read", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			b := make([]byte, 128)
			conn := &dialerTTLWrapperConn{
				Conn: &mocks.Conn{
					MockRead: func(b []byte) (int, error) {
						return len(b), nil
					},
				},
			}
			count, err := conn.Read(b)
			if err != nil {
				t.Fatal(err)
			}
			if count != len(b) {
				t.Fatal("unexpected count")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			b := make([]byte, 128)
			expectedErr := io.EOF
			conn := &dialerTTLWrapperConn{
				Conn: &mocks.Conn{
					MockRead: func(b []byte) (int, error) {
						return 0, expectedErr
					},
				},
			}
			count, err := conn.Read(b)
			if err == nil || err.Error() != netxlite.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("unexpected count")
			}
		})
	})

	t.Run("Write", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			b := make([]byte, 128)
			conn := &dialerTTLWrapperConn{
				Conn: &mocks.Conn{
					MockWrite: func(b []byte) (int, error) {
						return len(b), nil
					},
				},
			}
			count, err := conn.Write(b)
			if err != nil {
				t.Fatal(err)
			}
			if count != len(b) {
				t.Fatal("unexpected count")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			b := make([]byte, 128)
			expectedErr := io.EOF
			conn := &dialerTTLWrapperConn{
				Conn: &mocks.Conn{
					MockWrite: func(b []byte) (int, error) {
						return 0, expectedErr
					},
				},
			}
			count, err := conn.Write(b)
			if err == nil || err.Error() != netxlite.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("unexpected count")
			}
		})
	})

	t.Run("Close", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			conn := &dialerTTLWrapperConn{
				Conn: &mocks.Conn{
					MockClose: func() error {
						return nil
					},
				},
			}
			err := conn.Close()
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expectedErr := io.EOF
			conn := &dialerTTLWrapperConn{
				Conn: &mocks.Conn{
					MockClose: func() error {
						return expectedErr
					},
				},
			}
			err := conn.Close()
			if err == nil || err.Error() != netxlite.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
		})
	})
}

func TestSetTTL(t *testing.T) {
	d := NewDialerTTLWrapper()
	ctx := context.Background()
	conn, err := d.DialContext(ctx, "tcp", "1.1.1.1:80")
	if err != nil {
		t.Fatal("expected non-nil conn")
	}
	// test TTL set
	err = setTTL(conn, 1)
	if err != nil {
		t.Fatal("unexpected error in setting TTL", err)
	}
	var buf [512]byte
	_, err = conn.Write([]byte("1111"))
	if err != nil {
		t.Fatal("error writing", err)
	}
	r, _ := conn.Read(buf[:])
	if r != 0 {
		t.Fatal("unexpected output of size", r)
	}
	setTTL(conn, 64) // reset TTL to ensure conn closes successfully
	conn.Close()
	_, err = conn.Read(buf[:])
	if err == nil || err.Error() != netxlite.FailureConnectionAlreadyClosed {
		t.Fatal("failed to reset TTL")
	}
}
