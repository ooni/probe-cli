package tracex

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDialerConnectObserver(t *testing.T) {
	saver := &Saver{}
	obs := &dialerConnectObserver{
		saver: saver,
	}
	dialer := &mocks.Dialer{}
	out := obs.WrapDialer(dialer)
	dialSaver := out.(*DialerSaver)
	if dialSaver.Dialer != dialer {
		t.Fatal("invalid dialer")
	}
	if dialSaver.Saver != saver {
		t.Fatal("invalid saver")
	}
}

func TestDialerSaver(t *testing.T) {
	t.Run("on failure", func(t *testing.T) {
		expected := errors.New("mocked error")
		saver := &Saver{}
		dlr := &DialerSaver{
			Dialer: &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
					return nil, expected
				},
			},
			Saver: saver,
		}
		conn, err := dlr.DialContext(context.Background(), "tcp", "www.google.com:443")
		if !errors.Is(err, expected) {
			t.Fatal("expected another error here")
		}
		if conn != nil {
			t.Fatal("expected nil conn here")
		}
		ev := saver.Read()
		if len(ev) != 1 {
			t.Fatal("expected a single event here")
		}
		if ev[0].Value().Address != "www.google.com:443" {
			t.Fatal("unexpected Address")
		}
		if ev[0].Value().Duration <= 0 {
			t.Fatal("unexpected Duration")
		}
		if ev[0].Value().Err != "unknown_failure: mocked error" {
			t.Fatal("unexpected Err")
		}
		if ev[0].Name() != netxlite.ConnectOperation {
			t.Fatal("unexpected Name")
		}
		if ev[0].Value().Proto != "tcp" {
			t.Fatal("unexpected Proto")
		}
		if !ev[0].Value().Time.Before(time.Now()) {
			t.Fatal("unexpected Time")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.Dialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		dialer := &DialerSaver{
			Dialer: child,
			Saver:  &Saver{},
		}
		dialer.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestDialerReadWriteObserver(t *testing.T) {
	saver := &Saver{}
	obs := &dialerReadWriteObserver{
		saver: saver,
	}
	dialer := &mocks.Dialer{}
	out := obs.WrapDialer(dialer)
	dialSaver := out.(*DialerConnSaver)
	if dialSaver.Dialer != dialer {
		t.Fatal("invalid dialer")
	}
	if dialSaver.Saver != saver {
		t.Fatal("invalid saver")
	}
}

func TestDialerConnSaver(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			saver := &Saver{}
			dlr := &DialerConnSaver{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						return nil, expected
					},
				},
				Saver: saver,
			}
			conn, err := dlr.DialContext(context.Background(), "tcp", "www.google.com:443")
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn here")
			}
		})

		t.Run("on success", func(t *testing.T) {
			origConn := &mocks.Conn{}
			saver := &Saver{}
			dlr := &DialerConnSaver{
				Dialer: &DialerSaver{
					Dialer: &mocks.Dialer{
						MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
							return origConn, nil
						},
					},
					Saver: saver,
				},
				Saver: saver,
			}
			conn, err := dlr.DialContext(context.Background(), "tcp", "www.google.com:443")
			if err != nil {
				t.Fatal("not the error we expected", err)
			}
			cw := conn.(*dialerConnWrapper)
			if cw.Conn != origConn {
				t.Fatal("unexpected conn")
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
		dialer := &DialerConnSaver{
			Dialer: child,
			Saver:  &Saver{},
		}
		dialer.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestDialerConnWrapper(t *testing.T) {
	t.Run("Read", func(t *testing.T) {
		baseConn := &mocks.Conn{
			MockRead: func(b []byte) (int, error) {
				return 0, io.EOF
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockString: func() string {
						return "www.google.com:443"
					},
					MockNetwork: func() string {
						return "tcp"
					},
				}
			},
		}
		saver := &Saver{}
		conn := &dialerConnWrapper{
			Conn:  baseConn,
			saver: saver,
		}
		data := make([]byte, 155)
		count, err := conn.Read(data)
		if !errors.Is(err, io.EOF) {
			t.Fatal("unexpected err", err)
		}
		if count != 0 {
			t.Fatal("unexpected count")
		}
		ev := saver.Read()
		if len(ev) != 1 {
			t.Fatal("expected a single event here")
		}
		if ev[0].Value().Address != "www.google.com:443" {
			t.Fatal("unexpected Address")
		}
		if ev[0].Value().Duration <= 0 {
			t.Fatal("unexpected Duration")
		}
		if ev[0].Value().Err != netxlite.FailureEOFError {
			t.Fatal("unexpected Err")
		}
		if ev[0].Name() != netxlite.ReadOperation {
			t.Fatal("unexpected Name")
		}
		if ev[0].Value().Proto != "tcp" {
			t.Fatal("unexpected Proto")
		}
		if !ev[0].Value().Time.Before(time.Now()) {
			t.Fatal("unexpected Time")
		}
	})

	t.Run("Write", func(t *testing.T) {
		baseConn := &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return 0, io.EOF
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockString: func() string {
						return "www.google.com:443"
					},
					MockNetwork: func() string {
						return "tcp"
					},
				}
			},
		}
		saver := &Saver{}
		conn := &dialerConnWrapper{
			Conn:  baseConn,
			saver: saver,
		}
		data := make([]byte, 155)
		count, err := conn.Write(data)
		if !errors.Is(err, io.EOF) {
			t.Fatal("unexpected err", err)
		}
		if count != 0 {
			t.Fatal("unexpected count")
		}
		ev := saver.Read()
		if len(ev) != 1 {
			t.Fatal("expected a single event here")
		}
		if ev[0].Value().Address != "www.google.com:443" {
			t.Fatal("unexpected Address")
		}
		if ev[0].Value().Duration <= 0 {
			t.Fatal("unexpected Duration")
		}
		if ev[0].Value().Err != netxlite.FailureEOFError {
			t.Fatal("unexpected Err")
		}
		if ev[0].Name() != netxlite.WriteOperation {
			t.Fatal("unexpected Name")
		}
		if ev[0].Value().Proto != "tcp" {
			t.Fatal("unexpected Proto")
		}
		if !ev[0].Value().Time.Before(time.Now()) {
			t.Fatal("unexpected Time")
		}
	})
}
