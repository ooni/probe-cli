package tlsmiddlebox

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"syscall"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
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
}

func TestSetTTL(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}
		d := NewDialerTTLWrapper()
		ctx := context.Background()
		conn, err := d.DialContext(ctx, "tcp", "1.1.1.1:80")
		if err != nil {
			t.Fatal("expected non-nil conn")
		}
		// test TTL set
		err = setConnTTL(conn, 1)
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
			t.Fatal("unexpected output size", r)
		}
		setConnTTL(conn, 64) // reset TTL to ensure conn closes successfully
		conn.Close()
		_, err = conn.Read(buf[:])
		if err == nil || err.Error() != netxlite.FailureConnectionAlreadyClosed {
			t.Fatal("failed to reset TTL")
		}
	})

	t.Run("failure case", func(t *testing.T) {
		conn := &mocks.Conn{}
		err := setConnTTL(conn, 1)
		if !errors.Is(err, errInvalidConnWrapper) {
			t.Fatal("unexpected error")
		}
	})
}

func TestGetSoErr(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer srvr.Close()
		URL, err := url.Parse(srvr.URL)
		runtimex.PanicOnError(err, "url.Parse failed")
		d := NewDialerTTLWrapper()
		ctx := context.Background()
		conn, err := d.DialContext(ctx, "tcp", URL.Host)
		if err != nil {
			t.Fatal(err)
		}
		errno, err := getSoErr(conn)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		if !errors.Is(errno, syscall.Errno(0)) {
			t.Fatal("unexpected errno")
		}
	})

	t.Run("failure case", func(t *testing.T) {
		conn := &mocks.Conn{}
		errno, err := getSoErr(conn)
		if !errors.Is(err, errInvalidConnWrapper) {
			t.Fatal("unexpected error")
		}
		if errno != nil {
			t.Fatal("expected nil errorno")
		}
	})
}
