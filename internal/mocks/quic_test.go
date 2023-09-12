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
	"github.com/quic-go/quic-go"
)

func TestQUICDialer(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		expected := errors.New("mocked error")
		qcd := &QUICDialer{
			MockDialContext: func(ctx context.Context, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
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

func TestQUICEarlyConnection(t *testing.T) {
	t.Run("AcceptStream", func(t *testing.T) {
		expected := errors.New("mocked error")
		qconn := &QUICEarlyConnection{
			MockAcceptStream: func(ctx context.Context) (quic.Stream, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		stream, err := qconn.AcceptStream(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("AcceptUniStream", func(t *testing.T) {
		expected := errors.New("mocked error")
		qconn := &QUICEarlyConnection{
			MockAcceptUniStream: func(ctx context.Context) (quic.ReceiveStream, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		stream, err := qconn.AcceptUniStream(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("OpenStream", func(t *testing.T) {
		expected := errors.New("mocked error")
		qconn := &QUICEarlyConnection{
			MockOpenStream: func() (quic.Stream, error) {
				return nil, expected
			},
		}
		stream, err := qconn.OpenStream()
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("OpenStreamSync", func(t *testing.T) {
		expected := errors.New("mocked error")
		qconn := &QUICEarlyConnection{
			MockOpenStreamSync: func(ctx context.Context) (quic.Stream, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		stream, err := qconn.OpenStreamSync(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("OpenUniStream", func(t *testing.T) {
		expected := errors.New("mocked error")
		qconn := &QUICEarlyConnection{
			MockOpenUniStream: func() (quic.SendStream, error) {
				return nil, expected
			},
		}
		stream, err := qconn.OpenUniStream()
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("OpenUniStreamSync", func(t *testing.T) {
		expected := errors.New("mocked error")
		qconn := &QUICEarlyConnection{
			MockOpenUniStreamSync: func(ctx context.Context) (quic.SendStream, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		stream, err := qconn.OpenUniStreamSync(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("LocalAddr", func(t *testing.T) {
		qconn := &QUICEarlyConnection{
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
		qconn := &QUICEarlyConnection{
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
		qconn := &QUICEarlyConnection{
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
		qconn := &QUICEarlyConnection{
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
		state := quic.ConnectionState{SupportsDatagrams: true}
		qconn := &QUICEarlyConnection{
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
		qconn := &QUICEarlyConnection{
			MockHandshakeComplete: func() <-chan struct{} {
				return ctx.Done()
			},
		}
		out := qconn.HandshakeComplete()
		if !reflect.DeepEqual(ctx.Done(), out) {
			t.Fatal("not the channel we expected")
		}
	})

	t.Run("NextConnection", func(t *testing.T) {
		next := &QUICEarlyConnection{}
		qconn := &QUICEarlyConnection{
			MockNextConnection: func() quic.Connection {
				return next
			},
		}
		out := qconn.NextConnection()
		if !reflect.DeepEqual(next, out) {
			t.Fatal("not the context we expected")
		}
	})

	t.Run("SendMessage", func(t *testing.T) {
		expected := errors.New("mocked error")
		qconn := &QUICEarlyConnection{
			MockSendMessage: func(b []byte) error {
				return expected
			},
		}
		b := make([]byte, 17)
		err := qconn.SendMessage(b)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("ReceiveMessage", func(t *testing.T) {
		expected := errors.New("mocked error")
		ctx := context.Background()
		qconn := &QUICEarlyConnection{
			MockReceiveMessage: func(ctx context.Context) ([]byte, error) {
				return nil, expected
			},
		}
		b, err := qconn.ReceiveMessage(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if b != nil {
			t.Fatal("expected nil buffer here")
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
