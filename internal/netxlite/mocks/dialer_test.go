package mocks

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestDialer(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		expected := errors.New("mocked error")
		d := Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		conn, err := d.DialContext(ctx, "tcp", "8.8.8.8:53")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		d := &Dialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		d.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestConn(t *testing.T) {
	t.Run("Read", func(t *testing.T) {
		expected := errors.New("mocked error")
		c := &Conn{
			MockRead: func(b []byte) (int, error) {
				return 0, expected
			},
		}
		count, err := c.Read(make([]byte, 128))
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if count != 0 {
			t.Fatal("expected 0 bytes")
		}
	})

	t.Run("Write", func(t *testing.T) {
		expected := errors.New("mocked error")
		c := &Conn{
			MockWrite: func(b []byte) (int, error) {
				return 0, expected
			},
		}
		count, err := c.Write(make([]byte, 128))
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if count != 0 {
			t.Fatal("expected 0 bytes")
		}
	})

	t.Run("Close", func(t *testing.T) {
		expected := errors.New("mocked error")
		c := &Conn{
			MockClose: func() error {
				return expected
			},
		}
		err := c.Close()
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("LocalAddr", func(t *testing.T) {
		expected := &net.TCPAddr{
			IP:   net.IPv6loopback,
			Port: 1234,
		}
		c := &Conn{
			MockLocalAddr: func() net.Addr {
				return expected
			},
		}
		out := c.LocalAddr()
		if diff := cmp.Diff(expected, out); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("RemoteAddr", func(t *testing.T) {
		expected := &net.TCPAddr{
			IP:   net.IPv6loopback,
			Port: 1234,
		}
		c := &Conn{
			MockRemoteAddr: func() net.Addr {
				return expected
			},
		}
		out := c.RemoteAddr()
		if diff := cmp.Diff(expected, out); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("SetDeadline", func(t *testing.T) {
		expected := errors.New("mocked error")
		c := &Conn{
			MockSetDeadline: func(t time.Time) error {
				return expected
			},
		}
		err := c.SetDeadline(time.Time{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("SetReadDeadline", func(t *testing.T) {
		expected := errors.New("mocked error")
		c := &Conn{
			MockSetReadDeadline: func(t time.Time) error {
				return expected
			},
		}
		err := c.SetReadDeadline(time.Time{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("SetWriteDeadline", func(t *testing.T) {
		expected := errors.New("mocked error")
		c := &Conn{
			MockSetWriteDeadline: func(t time.Time) error {
				return expected
			},
		}
		err := c.SetWriteDeadline(time.Time{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})
}
