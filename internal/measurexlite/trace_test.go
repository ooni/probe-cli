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
	"github.com/ooni/probe-cli/v3/internal/testingx"
	"github.com/quic-go/quic-go"
	utls "gitlab.com/yawning/utls.git"
)

func TestNewTrace(t *testing.T) {
	t.Run("NewTrace correctly constructs a trace", func(t *testing.T) {
		const index = 17
		zeroTime := time.Now()
		trace := NewTrace(index, zeroTime)

		t.Run("Index", func(t *testing.T) {
			if trace.Index != index {
				t.Fatal("invalid index")
			}
		})

		t.Run("NetworkEvent has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalNetworkEvent{}
				ff.Fill(ev)
				select {
				case trace.networkEvent <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != NetworkEventBufferSize {
				t.Fatal("invalid NetworkEvent channel buffer size")
			}
		})

		t.Run("Netx is an instance of *netxlite.Netx with a nil .Underlying", func(t *testing.T) {
			if trace.Netx == nil {
				t.Fatal("expected non-nil .Netx")
			}
			netx, good := trace.Netx.(*netxlite.Netx)
			if !good {
				t.Fatal("not a *netxlite.Netx")
			}
			if netx.Underlying != nil {
				t.Fatal(".Underlying is not nil")
			}
		})

		t.Run("dnsLookup has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalDNSLookupResult{}
				ff.Fill(ev)
				select {
				case trace.dnsLookup <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != DNSLookupBufferSize {
				t.Fatal("invalid dnsLookup channel buffer size")
			}
		})

		t.Run("delayedDNSResponse has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalDNSLookupResult{}
				ff.Fill(ev)
				select {
				case trace.delayedDNSResponse <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != DelayedDNSResponseBufferSize {
				t.Fatal("invalid delayedDNSResponse channel buffer size")
			}
		})

		t.Run("tcpConnect has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalTCPConnectResult{}
				ff.Fill(ev)
				select {
				case trace.tcpConnect <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != TCPConnectBufferSize {
				t.Fatal("invalid tcpConnect channel buffer size")
			}
		})

		t.Run("tlsHandshake has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalTLSOrQUICHandshakeResult{}
				ff.Fill(ev)
				select {
				case trace.tlsHandshake <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != TLSHandshakeBufferSize {
				t.Fatal("invalid tlsHandshake channel buffer size")
			}
		})

		t.Run("quicHandshake has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalTLSOrQUICHandshakeResult{}
				ff.Fill(ev)
				select {
				case trace.quicHandshake <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != QUICHandshakeBufferSize {
				t.Fatal("invalid quicHandshake channel buffer size")
			}
		})

		t.Run("TimeNowFn is nil", func(t *testing.T) {
			if trace.timeNowFn != nil {
				t.Fatal("expected nil TimeNowFn")
			}
		})

		t.Run("ZeroTime", func(t *testing.T) {
			if !trace.ZeroTime.Equal(zeroTime) {
				t.Fatal("invalid zero time")
			}
		})
	})
}

func TestTrace(t *testing.T) {
	t.Run("NewStdlibResolver works as intended", func(t *testing.T) {
		t.Run("when nil", func(t *testing.T) {
			tx := NewTrace(0, time.Now())
			resolver := tx.NewStdlibResolver(model.DiscardLogger)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			addrs, err := resolver.LookupHost(ctx, "example.com")
			if err == nil || err.Error() != netxlite.FailureInterrupted {
				t.Fatal("unexpected err", err)
			}
			if len(addrs) != 0 {
				t.Fatal("expected array of size 0")
			}
		})
	})

	t.Run("NewParallelUDPResolver works as intended", func(t *testing.T) {
		tx := NewTrace(0, time.Now())
		dialer := netxlite.NewDialerWithoutResolver(model.DiscardLogger)
		resolver := tx.NewParallelUDPResolver(model.DiscardLogger, dialer, "1.1.1.1:53")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		addrs, err := resolver.LookupHost(ctx, "example.com")
		if err == nil || err.Error() != netxlite.FailureInterrupted {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 0 {
			t.Fatal("expected array of size 0")
		}
	})

	t.Run("NewParallelDNSOverHTTPSResolver works as intended", func(t *testing.T) {
		tx := NewTrace(0, time.Now())
		resolver := tx.NewParallelDNSOverHTTPSResolver(model.DiscardLogger, "https://dns.google.com")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		addrs, err := resolver.LookupHost(ctx, "example.com")
		if err == nil || err.Error() != netxlite.FailureInterrupted {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 0 {
			t.Fatal("expected array of size 0")
		}
	})

	t.Run("NewDialerWithoutResolver works as intended", func(t *testing.T) {
		tx := NewTrace(0, time.Now())
		dialer := tx.NewDialerWithoutResolver(model.DiscardLogger)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // fail immediately
		conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
		if err == nil || err.Error() != netxlite.FailureInterrupted {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("NewTLSHandshakerStdlib works as intended", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		tx := NewTrace(0, time.Now())
		thx := tx.NewTLSHandshakerStdlib(model.DiscardLogger)
		tcpConn := &mocks.Conn{
			MockSetDeadline: func(t time.Time) error {
				return nil
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockNetwork: func() string {
						return "tcp"
					},
					MockString: func() string {
						return "1.1.1.1:443"
					},
				}
			},
			MockWrite: func(b []byte) (int, error) {
				return 0, mockedErr
			},
			MockClose: func() error {
				return nil
			},
		}
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		ctx := context.Background()
		conn, err := thx.Handshake(ctx, tcpConn, tlsConfig)
		if !errors.Is(err, mockedErr) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("NewTLSHandshakerUTLS works as intended", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		tx := NewTrace(0, time.Now())
		thx := tx.NewTLSHandshakerUTLS(model.DiscardLogger, &utls.HelloGolang)
		tcpConn := &mocks.Conn{
			MockSetDeadline: func(t time.Time) error {
				return nil
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockNetwork: func() string {
						return "tcp"
					},
					MockString: func() string {
						return "1.1.1.1:443"
					},
				}
			},
			MockWrite: func(b []byte) (int, error) {
				return 0, mockedErr
			},
			MockClose: func() error {
				return nil
			},
		}
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		ctx := context.Background()
		conn, err := thx.Handshake(ctx, tcpConn, tlsConfig)
		if !errors.Is(err, mockedErr) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("NewQUICDialerWithoutResolver works as intended", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		tx := NewTrace(0, time.Now())
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
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		dialer := tx.NewQUICDialerWithoutResolver(listener, model.DiscardLogger)
		ctx := context.Background()
		qconn, err := dialer.DialContext(ctx, "1.1.1.1:443", tlsConfig, &quic.Config{})
		if !errors.Is(err, mockedErr) {
			t.Fatal("unexpected err", err)
		}
		if qconn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("TimeNowFn works as intended", func(t *testing.T) {
		fixedTime := time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
		tx := &Trace{
			timeNowFn: func() time.Time {
				return fixedTime
			},
		}
		if !tx.TimeNow().Equal(fixedTime) {
			t.Fatal("we cannot override time.Now calls")
		}
	})

	t.Run("TimeSince works as intended", func(t *testing.T) {
		t0 := time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
		t1 := t0.Add(10 * time.Second)
		tx := &Trace{
			timeNowFn: func() time.Time {
				return t1
			},
		}
		if tx.TimeSince(t0) != 10*time.Second {
			t.Fatal("apparently Trace.Since is broken")
		}
	})
}

func TestTags(t *testing.T) {
	trace := NewTrace(0, time.Now(), "antani")
	got := trace.Tags()
	if diff := cmp.Diff([]string{"antani"}, got); diff != "" {
		t.Fatal(diff)
	}
}
