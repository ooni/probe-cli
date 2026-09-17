package mocks

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/quic-go/quic-go"
)

func TestQUICDialer(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		expected := errors.New("mocked error")
		qcd := &QUICDialer{
			MockDialContext: func(ctx context.Context, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (model.QUICConn, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		tlsConfig := &tls.Config{}
		quicConfig := &quic.Config{}
		qconn, err := qcd.DialContext(ctx, "dns.google:443", tlsConfig, quicConfig)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if qconn != nil {
			t.Fatal("expected nil connection")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		qcd := &QUICDialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		qcd.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestQUICConn(t *testing.T) {
	t.Run("LocalAddr", func(t *testing.T) {
		qconn := &QUICConn{
			MockLocalAddr: func() net.Addr {
				return &net.UDPAddr{}
			},
		}
		addr := qconn.LocalAddr()
		if !reflect.ValueOf(addr).Elem().IsZero() {
			t.Fatal("expected a zero address here")
		}
	})

	t.Run("RemoteAddr", func(t *testing.T) {
		qconn := &QUICConn{
			MockRemoteAddr: func() net.Addr {
				return &net.UDPAddr{}
			},
		}
		addr := qconn.RemoteAddr()
		if !reflect.ValueOf(addr).Elem().IsZero() {
			t.Fatal("expected a zero address here")
		}
	})

	t.Run("CloseWithError", func(t *testing.T) {
		expected := errors.New("mocked error")
		qconn := &QUICConn{
			MockCloseWithError: func(
				code quic.ApplicationErrorCode, reason string) error {
				return expected
			},
		}
		err := qconn.CloseWithError(0, "")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("Context", func(t *testing.T) {
		ctx := context.Background()
		qconn := &QUICConn{
			MockContext: func() context.Context {
				return ctx
			},
		}
		out := qconn.Context()
		if !reflect.DeepEqual(ctx, out) {
			t.Fatal("not the context we expected")
		}
	})

	t.Run("ConnectionState", func(t *testing.T) {
		state := quic.ConnectionState{Used0RTT: true}
		qconn := &QUICConn{
			MockConnectionState: func() quic.ConnectionState {
				return state
			},
		}
		out := qconn.ConnectionState()
		if !reflect.DeepEqual(state, out) {
			t.Fatal("not the context we expected")
		}
	})

	t.Run("HandshakeComplete", func(t *testing.T) {
		ctx := context.Background()
		qconn := &QUICConn{
			MockHandshakeComplete: func() <-chan struct{} {
				return ctx.Done()
			},
		}
		out := qconn.HandshakeComplete()
		if !reflect.DeepEqual(ctx.Done(), out) {
			t.Fatal("not the channel we expected")
		}
	})
}

func TestQUICUDPLikeConn(t *testing.T) {
	t.Run("WriteTo", func(t *testing.T) {
		expected := errors.New("mocked error")
		quc := &UDPLikeConn{
			MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
				return 0, expected
			},
		}
		pkt := make([]byte, 128)
		addr := &net.UDPAddr{}
		cnt, err := quc.WriteTo(pkt, addr)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if cnt != 0 {
			t.Fatal("expected zero here")
		}
	})

	t.Run("ConnClose", func(t *testing.T) {
		expected := errors.New("mocked error")
		quc := &UDPLikeConn{
			MockClose: func() error {
				return expected
			},
		}
		err := quc.Close()
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("LocalAddr", func(t *testing.T) {
		expected := &net.TCPAddr{
			IP:   net.IPv6loopback,
			Port: 1234,
		}
		c := &UDPLikeConn{
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
		c := &UDPLikeConn{
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
		c := &UDPLikeConn{
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
		c := &UDPLikeConn{
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
		c := &UDPLikeConn{
			MockSetWriteDeadline: func(t time.Time) error {
				return expected
			},
		}
		err := c.SetWriteDeadline(time.Time{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("ConnReadFrom", func(t *testing.T) {
		expected := errors.New("mocked error")
		quc := &UDPLikeConn{
			MockReadFrom: func(b []byte) (int, net.Addr, error) {
				return 0, nil, expected
			},
		}
		b := make([]byte, 128)
		n, addr, err := quc.ReadFrom(b)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if n != 0 {
			t.Fatal("expected zero here")
		}
		if addr != nil {
			t.Fatal("expected nil here")
		}
	})

	t.Run("SyscallConn", func(t *testing.T) {
		expected := errors.New("mocked error")
		quc := &UDPLikeConn{
			MockSyscallConn: func() (syscall.RawConn, error) {
				return nil, expected
			},
		}
		conn, err := quc.SyscallConn()
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if conn != nil {
			t.Fatal("expected nil here")
		}
	})

	t.Run("SetReadBuffer", func(t *testing.T) {
		expected := errors.New("mocked error")
		quc := &UDPLikeConn{
			MockSetReadBuffer: func(n int) error {
				return expected
			},
		}
		err := quc.SetReadBuffer(1 << 10)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})
}
