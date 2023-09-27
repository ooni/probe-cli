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
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quictesting"
	"github.com/ooni/probe-cli/v3/internal/testingx"
	"github.com/quic-go/quic-go"
)

func TestNewQUICDialerWithoutResolver(t *testing.T) {
	t.Run("NewQUICDialerWithoutResolver creates a wrapped dialer", func(t *testing.T) {
		underlying := &mocks.QUICDialer{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.Netx = &mocks.MeasuringNetwork{
			MockNewQUICDialerWithoutResolver: func(listener model.UDPListener, logger model.DebugLogger, w ...model.QUICDialerWrapper) model.QUICDialer {
				return underlying
			},
		}
		listener := &mocks.UDPListener{}
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
			MockDialContext: func(ctx context.Context, address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlyConnection, error) {
				gotTrace := netxlite.ContextTraceOrDefault(ctx)
				hasCorrectTrace = (gotTrace == trace)
				return nil, expectedErr
			},
		}
		trace.Netx = &mocks.MeasuringNetwork{
			MockNewQUICDialerWithoutResolver: func(listener model.UDPListener, logger model.DebugLogger, w ...model.QUICDialerWrapper) model.QUICDialer {
				return underlying
			},
		}
		listener := &mocks.UDPListener{}
		dialer := trace.NewQUICDialerWithoutResolver(listener, model.DiscardLogger)
		ctx := context.Background()
		conn, err := dialer.DialContext(ctx, "1.1.1.1:443", &tls.Config{}, &quic.Config{})
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
		trace.Netx = &mocks.MeasuringNetwork{
			MockNewQUICDialerWithoutResolver: func(listener model.UDPListener, logger model.DebugLogger, w ...model.QUICDialerWrapper) model.QUICDialer {
				return underlying
			},
		}
		listener := &mocks.UDPListener{}
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
		trace := NewTrace(0, zeroTime, "antani")
		trace.timeNowFn = td.Now // deterministic time tracking
		pconn := &mocks.UDPLikeConn{
			MockLocalAddr: func() net.Addr {
				return &net.UDPAddr{
					// quic-go does not allow the use of the same net.PacketConn for multiple "Dial"
					// calls (unless a quic.Transport is used), so we have to make sure to mock local
					// addresses with different ports, as tests run in parallel.
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
			MockSetReadBuffer: func(n int) error {
				return nil
			},
		}
		listener := &mocks.UDPListener{
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
		qconn, err := dialer.DialContext(ctx, "1.1.1.1:443", tlsConfig, &quic.Config{})
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
				Network:            "udp",
				Address:            "1.1.1.1:443",
				CipherSuite:        "",
				Failure:            &expectedFailure,
				NegotiatedProtocol: "",
				NoTLSVerify:        true,
				PeerCertificates:   []model.ArchivalMaybeBinaryData{},
				ServerName:         "dns.cloudflare.com",
				T:                  time.Second.Seconds(),
				Tags:               []string{"antani"},
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
					Tags:      []string{"antani"},
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
					T0:        time.Second.Seconds(),
					T:         time.Second.Seconds(),
					Tags:      []string{"antani"},
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
					// quic-go does not allow the use of the same net.PacketConn for multiple "Dial"
					// calls (unless a quic.Transport is used), so we have to make sure to mock local
					// addresses with different ports, as tests run in parallel.
					Port: 1,
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
			MockSetReadBuffer: func(n int) error {
				return nil
			},
		}
		listener := &mocks.UDPListener{
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
		qconn, err := dialer.DialContext(ctx, "1.1.1.1:443", tlsConfig, &quic.Config{})
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

func TestOnQUICHandshakeDoneExtractsTheConnectionState(t *testing.T) {
	// create a trace
	trace := NewTrace(0, time.Now())

	// create a QUIC dialer
	udpListener := netxlite.NewUDPListener()
	quicDialer := trace.NewQUICDialerWithoutResolver(udpListener, model.DiscardLogger)

	// dial with the endpoint we use for testing
	quicConn, err := quicDialer.DialContext(
		context.Background(),
		quictesting.Endpoint("443"),
		&tls.Config{
			InsecureSkipVerify: true,
		},
		&quic.Config{},
	)
	defer MaybeCloseQUICConn(quicConn)

	// we do not expect to see an error here
	if err != nil {
		t.Fatal(err)
	}

	// extract the QUIC handshake event
	event := trace.FirstQUICHandshakeOrNil()
	if event == nil {
		t.Fatal("expected non-nil event")
	}

	// make sure we have parsed the QUIC connection state
	if event.NegotiatedProtocol != "h3" {
		t.Fatal("it seems we did not parse the QUIC connection state")
	}
}

func TestFirstQUICHandshake(t *testing.T) {
	t.Run("returns nil when buffer is empty", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		got := trace.FirstQUICHandshakeOrNil()
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
			Network:            "udp",
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
			Network:            "udp",
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
		got := trace.FirstQUICHandshakeOrNil()
		if diff := cmp.Diff(got, expect[0]); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestMaybeCloseQUICConn(t *testing.T) {
	type closeQuicTest struct {
		name   string
		input  quic.EarlyConnection
		called bool
	}
	var called bool

	tests := []closeQuicTest{
		{
			name:   "with nil earlyconn",
			input:  nil,
			called: false,
		},
		{
			name: "with nonnil conn",
			input: &mocks.QUICEarlyConnection{
				MockCloseWithError: func(code quic.ApplicationErrorCode, reason string) error {
					called = true
					return nil
				},
			},
			called: true,
		},
	}
	for _, test := range tests {
		err := MaybeCloseQUICConn(test.input)
		if err != nil {
			t.Fatalf("MaybeCloseQUICConn: unexpected failure (%s)", test.name)
		}
		if called != test.called {
			t.Fatalf("MaybeCloseQUICConn: unexpected behavior (%s)", test.name)
		}
		called = false
	}
}
