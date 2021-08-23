package netxmocks

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
	"github.com/ooni/probe-cli/v3/internal/quicx"
)

func TestQUICListenerListen(t *testing.T) {
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
}

func TestQUICContextDialerDialContext(t *testing.T) {
	expected := errors.New("mocked error")
	qcd := &QUICContextDialer{
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
}

func TestQUICEarlySessionAcceptStream(t *testing.T) {
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
}

func TestQUICEarlySessionAcceptUniStream(t *testing.T) {
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
}

func TestQUICEarlySessionOpenStream(t *testing.T) {
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
}

func TestQUICEarlySessionOpenStreamSync(t *testing.T) {
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
}

func TestQUICEarlySessionOpenUniStream(t *testing.T) {
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
}

func TestQUICEarlySessionOpenUniStreamSync(t *testing.T) {
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
}

func TestQUICEarlySessionLocalAddr(t *testing.T) {
	sess := &QUICEarlySession{
		MockLocalAddr: func() net.Addr {
			return &net.UDPAddr{}
		},
	}
	addr := sess.LocalAddr()
	if !reflect.ValueOf(addr).Elem().IsZero() {
		t.Fatal("expected a zero address here")
	}
}

func TestQUICEarlySessionRemoteAddr(t *testing.T) {
	sess := &QUICEarlySession{
		MockRemoteAddr: func() net.Addr {
			return &net.UDPAddr{}
		},
	}
	addr := sess.RemoteAddr()
	if !reflect.ValueOf(addr).Elem().IsZero() {
		t.Fatal("expected a zero address here")
	}
}

func TestQUICEarlySessionCloseWithError(t *testing.T) {
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
}

func TestQUICEarlySessionContext(t *testing.T) {
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
}

func TestQUICEarlySessionConnectionState(t *testing.T) {
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
}

func TestQUICEarlySessionHandshakeComplete(t *testing.T) {
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
}

func TestQUICEarlySessionNextSession(t *testing.T) {
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
}

func TestQUICEarlySessionSendMessage(t *testing.T) {
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
}

func TestQUICEarlySessionReceiveMessage(t *testing.T) {
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
}

func TestQUICUDPConnWriteTo(t *testing.T) {
	expected := errors.New("mocked error")
	quc := &QUICUDPConn{
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
}

func TestQUICUDPConnClose(t *testing.T) {
	expected := errors.New("mocked error")
	quc := &QUICUDPConn{
		MockClose: func() error {
			return expected
		},
	}
	err := quc.Close()
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}

func TestQUICUDPConnLocalAddrWorks(t *testing.T) {
	expected := &net.TCPAddr{
		IP:   net.IPv6loopback,
		Port: 1234,
	}
	c := &QUICUDPConn{
		MockLocalAddr: func() net.Addr {
			return expected
		},
	}
	out := c.LocalAddr()
	if diff := cmp.Diff(expected, out); diff != "" {
		t.Fatal(diff)
	}
}

func TestQUICUDPConnRemoteAddrWorks(t *testing.T) {
	expected := &net.TCPAddr{
		IP:   net.IPv6loopback,
		Port: 1234,
	}
	c := &QUICUDPConn{
		MockRemoteAddr: func() net.Addr {
			return expected
		},
	}
	out := c.RemoteAddr()
	if diff := cmp.Diff(expected, out); diff != "" {
		t.Fatal(diff)
	}
}

func TestQUICUDPConnSetDeadline(t *testing.T) {
	expected := errors.New("mocked error")
	c := &QUICUDPConn{
		MockSetDeadline: func(t time.Time) error {
			return expected
		},
	}
	err := c.SetDeadline(time.Time{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}

func TestQUICUDPConnSetReadDeadline(t *testing.T) {
	expected := errors.New("mocked error")
	c := &QUICUDPConn{
		MockSetReadDeadline: func(t time.Time) error {
			return expected
		},
	}
	err := c.SetReadDeadline(time.Time{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}

func TestQUICUDPConnSetWriteDeadline(t *testing.T) {
	expected := errors.New("mocked error")
	c := &QUICUDPConn{
		MockSetWriteDeadline: func(t time.Time) error {
			return expected
		},
	}
	err := c.SetWriteDeadline(time.Time{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}

func TestQUICUDPConnReadFrom(t *testing.T) {
	expected := errors.New("mocked error")
	quc := &QUICUDPConn{
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
}

func TestQUICUDPConnSyscallConn(t *testing.T) {
	expected := errors.New("mocked error")
	quc := &QUICUDPConn{
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
}

func TestQUICUDPConnSetReadBuffer(t *testing.T) {
	expected := errors.New("mocked error")
	quc := &QUICUDPConn{
		MockSetReadBuffer: func(n int) error {
			return expected
		},
	}
	err := quc.SetReadBuffer(1 << 10)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}
