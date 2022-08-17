package measurexlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNewQUICListener(t *testing.T) {
	t.Run("NewQUICListenerTrace creates a wrapped listener", func(t *testing.T) {
		underlying := &mocks.QUICListener{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		listenert := trace.WrapQUICListener(underlying).(*quicListenerTrace)
		if listenert.QUICListener != underlying {
			t.Fatal("invalid quic dialer")
		}
		if listenert.tx != trace {
			t.Fatal("invalid trace")
		}
	})

	t.Run("Listen works as intended", func(t *testing.T) {
		t.Run("with error", func(t *testing.T) {
			zeroTime := time.Now()
			trace := NewTrace(0, zeroTime)
			mockedErr := errors.New("mocked")
			mockListener := &mocks.QUICListener{
				MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
					return nil, mockedErr
				},
			}
			listener := trace.WrapQUICListener(mockListener)
			pconn, err := listener.Listen(&net.UDPAddr{})
			if !errors.Is(err, mockedErr) {
				t.Fatal("unexpected err", err)
			}
			if pconn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("without error", func(t *testing.T) {
			zeroTime := time.Now()
			trace := NewTrace(0, zeroTime)
			mockConn := &mocks.UDPLikeConn{}
			mockListener := &mocks.QUICListener{
				MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
					return mockConn, nil
				},
			}
			listener := trace.WrapQUICListener(mockListener)
			pconn, err := listener.Listen(&net.UDPAddr{})
			if err != nil {
				t.Fatal("unexpected err", err)
			}
			conn := pconn.(*udpLikeConnTrace)
			if conn.UDPLikeConn != mockConn {
				t.Fatal("invalid conn")
			}
			if conn.tx != trace {
				t.Fatal("invalid trace")
			}
		})
	})
}

func TestNewQUICDialerWithoutResolver(t *testing.T) {
	t.Run("NewQUICDialerWithoutResolver creates a wrapped dialer", func(t *testing.T) {
		underlying := &mocks.QUICDialer{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NewQUICDialerWithoutResolverFn = func(listener model.QUICListener, dl model.DebugLogger) model.QUICDialer {
			return underlying
		}
		listener := &mocks.QUICListener{}
		dialer := trace.NewQUICDialerWithoutResolver(listener, model.DiscardLogger)
		dt := dialer.(*quicDialerTrace)
		if dt.qd != underlying {
			t.Fatal("invalid quic dialer")
		}
		if dt.tx != trace {
			t.Fatal("invalid trace")
		}
	})

	t.Run("DialContext calls the underlying dialer with context-based tracing", func(t *testing.T) {
		expectedErr := errors.New("mocked err")
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		var hasCorrectTrace bool
		underlying := &mocks.QUICDialer{
			MockDialContext: func(ctx context.Context, network, address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlyConnection, error) {
				gotTrace := netxlite.ContextTraceOrDefault(ctx)
				hasCorrectTrace = (gotTrace == trace)
				return nil, expectedErr
			},
		}
		trace.NewQUICDialerWithoutResolverFn = func(listener model.QUICListener, dl model.DebugLogger) model.QUICDialer {
			return underlying
		}
		listener := &mocks.QUICListener{}
		dialer := trace.NewQUICDialerWithoutResolver(listener, model.DiscardLogger)
		ctx := context.Background()
		conn, err := dialer.DialContext(ctx, "udp", "1.1.1.1:443", &tls.Config{}, &quic.Config{})
		if !errors.Is(err, expectedErr) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		if !hasCorrectTrace {
			t.Fatal("does not have the correct trace")
		}
	})

	t.Run("CloseIdleConnection is correctly forwarded", func(t *testing.T) {
		var called bool
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		underlying := &mocks.QUICDialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		trace.NewQUICDialerWithoutResolverFn = func(listener model.QUICListener, dl model.DebugLogger) model.QUICDialer {
			return underlying
		}
		listener := &mocks.QUICListener{}
		dialer := trace.NewQUICDialerWithoutResolver(listener, model.DiscardLogger)
		dialer.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("DialContext saves into trace", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = td.Now // deterministic time tracking
		pconn := &mocks.UDPLikeConn{
			MockLocalAddr: func() net.Addr {
				return &net.UDPAddr{
					Port: 0,
				}
			},
			MockRemoteAddr: func() net.Addr {
				return &net.UDPAddr{
					Port: 0,
				}
			},
			MockSyscallConn: func() (syscall.RawConn, error) {
				return nil, mockedErr
			},
			MockClose: func() error {
				return nil
			},
		}
		listener := &mocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return pconn, nil
			},
		}
		dialer := trace.NewQUICDialerWithoutResolver(listener, model.DiscardLogger)
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "dns.cloudflare.com",
		}
		ctx := context.Background()
		qconn, err := dialer.DialContext(ctx, "udp", "1.1.1.1:443", tlsConfig, &quic.Config{})
		if !errors.Is(err, mockedErr) {
			t.Fatal("unexpected err", err)
		}
		if qconn != nil {
			t.Fatal("expected nil qconn")
		}

		t.Run("QUICHandshake events", func(t *testing.T) {
			events := trace.QUICHandshakes()
			if len(events) != 1 {
				t.Fatal("expected to see single QUICHandshake event")
			}
			expectedFailure := "unknown_failure: mocked"
			expect := &model.ArchivalTLSOrQUICHandshakeResult{
				Network:            "quic",
				Address:            "1.1.1.1:443",
				CipherSuite:        "",
				Failure:            &expectedFailure,
				NegotiatedProtocol: "",
				NoTLSVerify:        true,
				PeerCertificates:   []model.ArchivalMaybeBinaryData{},
				ServerName:         "dns.cloudflare.com",
				T:                  time.Second.Seconds(),
				Tags:               []string{},
				TLSVersion:         "",
			}
			got := events[0]
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("Network events", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 2 {
				t.Fatal("expected to see three Network events")
			}

			t.Run("quic_handshake_start", func(t *testing.T) {
				expect := &model.ArchivalNetworkEvent{
					Address:   "",
					Failure:   nil,
					NumBytes:  0,
					Operation: "quic_handshake_start",
					Proto:     "",
					T:         0,
					Tags:      []string{},
				}
				got := events[0]
				if diff := cmp.Diff(expect, got); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("quic_handshake_done", func(t *testing.T) {
				expect := &model.ArchivalNetworkEvent{
					Address:   "",
					Failure:   nil,
					NumBytes:  0,
					Operation: "quic_handshake_done",
					Proto:     "",
					T:         time.Second.Seconds(),
					Tags:      []string{},
				}
				got := events[1]
				if diff := cmp.Diff(expect, got); diff != "" {
					t.Fatal(diff)
				}
			})
		})

	})

	t.Run("DialContext discards events when buffer is full", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.networkEvent = make(chan *model.ArchivalNetworkEvent)              // no buffer
		trace.quicHandshake = make(chan *model.ArchivalTLSOrQUICHandshakeResult) // no buffer
		pconn := &mocks.UDPLikeConn{
			MockLocalAddr: func() net.Addr {
				return &net.UDPAddr{
					Port: 0,
				}
			},
			MockRemoteAddr: func() net.Addr {
				return &net.UDPAddr{
					Port: 0,
				}
			},
			MockSyscallConn: func() (syscall.RawConn, error) {
				return nil, mockedErr
			},
			MockClose: func() error {
				return nil
			},
		}
		listener := &mocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return pconn, nil
			},
		}
		dialer := trace.NewQUICDialerWithoutResolver(listener, model.DiscardLogger)
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "dns.cloudflare.com",
		}
		ctx := context.Background()
		qconn, err := dialer.DialContext(ctx, "udp", "1.1.1.1:443", tlsConfig, &quic.Config{})
		if !errors.Is(err, mockedErr) {
			t.Fatal("unexpected err", err)
		}
		if qconn != nil {
			t.Fatal("expected nil qconn")
		}

		t.Run("QUiCHandshake events", func(t *testing.T) {
			events := trace.QUICHandshakes()
			if len(events) != 0 {
				t.Fatal("expected to see no QUICHandshake events")
			}
		})

		t.Run("Network events", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 0 {
				t.Fatal("expected to see no network events")
			}
		})
	})
}

func TestFirstQUICHandshake(t *testing.T) {
	t.Run("returns nil when buffer is empty", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		got := trace.FirstQUICHandshake()
		if got != nil {
			t.Fatal("expected nil event")
		}
	})

	t.Run("return first non-nil QUICHandshake", func(t *testing.T) {
		filler := func(tx *Trace, events []*model.ArchivalTLSOrQUICHandshakeResult) {
			for _, ev := range events {
				tx.quicHandshake <- ev
			}
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		expect := []*model.ArchivalTLSOrQUICHandshakeResult{{
			Network:            "quic",
			Address:            "1.1.1.1:443",
			CipherSuite:        "",
			Failure:            nil,
			NegotiatedProtocol: "",
			NoTLSVerify:        true,
			PeerCertificates:   []model.ArchivalMaybeBinaryData{},
			ServerName:         "dns.cloudflare.com",
			T:                  time.Second.Seconds(),
			Tags:               []string{},
			TLSVersion:         "",
		}, {
			Network:            "quic",
			Address:            "8.8.8.8:443",
			CipherSuite:        "",
			Failure:            nil,
			NegotiatedProtocol: "",
			NoTLSVerify:        true,
			PeerCertificates:   []model.ArchivalMaybeBinaryData{},
			ServerName:         "dns.google.com",
			T:                  time.Second.Seconds(),
			Tags:               []string{},
			TLSVersion:         "",
		}}
		filler(trace, expect)
		got := trace.FirstQUICHandshake()
		if diff := cmp.Diff(got, expect[0]); diff != "" {
			t.Fatal(diff)
		}
	})
}
