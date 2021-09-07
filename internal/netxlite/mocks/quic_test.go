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
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

func TestQUICListenerListen(t *testing.T) {
	t.Run("Listen", func(t *testing.T) {
		expected := errors.New("mocked error")
		ql := &QUICListener{
			MockListen: func(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
				return nil, expected
			},
		}
		pconn, err := ql.Listen(&net.UDPAddr{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", expected)
		}
		if pconn != nil {
			t.Fatal("expected nil conn here")
		}
	})
}

func TestQUICDialer(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		expected := errors.New("mocked error")
		qcd := &QUICDialer{
			MockDialContext: func(ctx context.Context, network string, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		tlsConfig := &tls.Config{}
		quicConfig := &quic.Config{}
		sess, err := qcd.DialContext(ctx, "udp", "dns.google:443", tlsConfig, quicConfig)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if sess != nil {
			t.Fatal("expected nil session")
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

func TestQUICEarlySession(t *testing.T) {
	t.Run("AcceptStream", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockAcceptStream: func(ctx context.Context) (quic.Stream, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		stream, err := sess.AcceptStream(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("AcceptUniStream", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockAcceptUniStream: func(ctx context.Context) (quic.ReceiveStream, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		stream, err := sess.AcceptUniStream(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("OpenStream", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockOpenStream: func() (quic.Stream, error) {
				return nil, expected
			},
		}
		stream, err := sess.OpenStream()
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("OpenStreamSync", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockOpenStreamSync: func(ctx context.Context) (quic.Stream, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		stream, err := sess.OpenStreamSync(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("OpenUniStream", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockOpenUniStream: func() (quic.SendStream, error) {
				return nil, expected
			},
		}
		stream, err := sess.OpenUniStream()
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("OpenUniStreamSync", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockOpenUniStreamSync: func(ctx context.Context) (quic.SendStream, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		stream, err := sess.OpenUniStreamSync(ctx)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if stream != nil {
			t.Fatal("expected nil stream here")
		}
	})

	t.Run("LocalAddr", func(t *testing.T) {
		sess := &QUICEarlySession{
			MockLocalAddr: func() net.Addr {
				return &net.UDPAddr{}
			},
		}
		addr := sess.LocalAddr()
		if !reflect.ValueOf(addr).Elem().IsZero() {
			t.Fatal("expected a zero address here")
		}
	})

	t.Run("RemoteAddr", func(t *testing.T) {
		sess := &QUICEarlySession{
			MockRemoteAddr: func() net.Addr {
				return &net.UDPAddr{}
			},
		}
		addr := sess.RemoteAddr()
		if !reflect.ValueOf(addr).Elem().IsZero() {
			t.Fatal("expected a zero address here")
		}
	})

	t.Run("CloseWithError", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockCloseWithError: func(
				code quic.ApplicationErrorCode, reason string) error {
				return expected
			},
		}
		err := sess.CloseWithError(0, "")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("Context", func(t *testing.T) {
		ctx := context.Background()
		sess := &QUICEarlySession{
			MockContext: func() context.Context {
				return ctx
			},
		}
		out := sess.Context()
		if !reflect.DeepEqual(ctx, out) {
			t.Fatal("not the context we expected")
		}
	})

	t.Run("ConnectionState", func(t *testing.T) {
		state := quic.ConnectionState{SupportsDatagrams: true}
		sess := &QUICEarlySession{
			MockConnectionState: func() quic.ConnectionState {
				return state
			},
		}
		out := sess.ConnectionState()
		if !reflect.DeepEqual(state, out) {
			t.Fatal("not the context we expected")
		}
	})

	t.Run("HandshakeComplete", func(t *testing.T) {
		ctx := context.Background()
		sess := &QUICEarlySession{
			MockHandshakeComplete: func() context.Context {
				return ctx
			},
		}
		out := sess.HandshakeComplete()
		if !reflect.DeepEqual(ctx, out) {
			t.Fatal("not the context we expected")
		}
	})

	t.Run("NextSession", func(t *testing.T) {
		next := &QUICEarlySession{}
		sess := &QUICEarlySession{
			MockNextSession: func() quic.Session {
				return next
			},
		}
		out := sess.NextSession()
		if !reflect.DeepEqual(next, out) {
			t.Fatal("not the context we expected")
		}
	})

	t.Run("SendMessage", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockSendMessage: func(b []byte) error {
				return expected
			},
		}
		b := make([]byte, 17)
		err := sess.SendMessage(b)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("ReceiveMessage", func(t *testing.T) {
		expected := errors.New("mocked error")
		sess := &QUICEarlySession{
			MockReceiveMessage: func() ([]byte, error) {
				return nil, expected
			},
		}
		b, err := sess.ReceiveMessage()
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
		quc := &QUICUDPLikeConn{
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
		quc := &QUICUDPLikeConn{
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
		c := &QUICUDPLikeConn{
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
		c := &QUICUDPLikeConn{
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
		c := &QUICUDPLikeConn{
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
		c := &QUICUDPLikeConn{
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
		c := &QUICUDPLikeConn{
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
		quc := &QUICUDPLikeConn{
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
		quc := &QUICUDPLikeConn{
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
		quc := &QUICUDPLikeConn{
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
