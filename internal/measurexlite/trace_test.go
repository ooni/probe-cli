package measurexlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
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
				case trace.NetworkEvent <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != NetworkEventBufferSize {
				t.Fatal("invalid NetworkEvent channel buffer size")
			}
		})

		t.Run("NewParallelUDPResolverFn is nil", func(t *testing.T) {
			if trace.NewParallelUDPResolverFn != nil {
				t.Fatal("expected nil NewParallelUDPResolverFn")
			}
		})

		t.Run("NewParallelDNSOverHTTPSResolverFn is nil", func(t *testing.T) {
			if trace.NewParallelDNSOverHTTPSResolverFn != nil {
				t.Fatal("expected nil NewParallelDNSOverHTTPSResolverFn")
			}
		})

		t.Run("NewDialerWithoutResolverFn is nil", func(t *testing.T) {
			if trace.NewDialerWithoutResolverFn != nil {
				t.Fatal("expected nil NewDialerWithoutResolverFn")
			}
		})

		t.Run("NewTLSHandshakerStdlibFn is nil", func(t *testing.T) {
			if trace.NewTLSHandshakerStdlibFn != nil {
				t.Fatal("expected nil NewTLSHandshakerStdlibFn")
			}
		})

		t.Run("DNSLookup has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalDNSLookupResult{}
				ff.Fill(ev)
				select {
				case trace.DNSLookup <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != DNSLookupBufferSize {
				t.Fatal("invalid DNSLookup channel buffer size")
			}
		})

		t.Run("TCPConnect has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalTCPConnectResult{}
				ff.Fill(ev)
				select {
				case trace.TCPConnect <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != TCPConnectBufferSize {
				t.Fatal("invalid TCPConnect channel buffer size")
			}
		})

		t.Run("TLSHandshake has the expected buffer size", func(t *testing.T) {
			ff := &testingx.FakeFiller{}
			var idx int
		Loop:
			for {
				ev := &model.ArchivalTLSOrQUICHandshakeResult{}
				ff.Fill(ev)
				select {
				case trace.TLSHandshake <- ev:
					idx++
				default:
					break Loop
				}
			}
			if idx != TLSHandshakeBufferSize {
				t.Fatal("invalid TLSHandshake channel buffer size")
			}
		})

		t.Run("TimeNowFn is nil", func(t *testing.T) {
			if trace.TimeNowFn != nil {
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
	t.Run("NewParallelUDPResolverFn works as intended", func(t *testing.T) {
		t.Run("when not nil", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			tx := &Trace{
				NewParallelUDPResolverFn: func(logger model.Logger, dialer model.Dialer, address string) model.Resolver {
					return &mocks.Resolver{
						MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
							return []string{}, mockedErr
						},
					}
				},
			}
			dialer := &mocks.Dialer{}
			resolver := tx.newParallelUDPResolver(model.DiscardLogger, dialer, "1.1.1.1:53")
			ctx := context.Background()
			addrs, err := resolver.LookupHost(ctx, "example.com")
			if !errors.Is(err, mockedErr) {
				t.Fatal("unexpected err", err)
			}
			if len(addrs) != 0 {
				t.Fatal("expected array of size 0")
			}
		})

		t.Run("when nil", func(t *testing.T) {
			tx := &Trace{
				NewParallelUDPResolverFn: nil,
			}
			dialer := netxlite.NewDialerWithoutResolver(model.DiscardLogger)
			resolver := tx.newParallelUDPResolver(model.DiscardLogger, dialer, "1.1.1.1:53")
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

	t.Run("NewParallelDNSOverHTTPSResolverFn works as intended", func(t *testing.T) {
		t.Run("when not nil", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			tx := &Trace{
				NewParallelDNSOverHTTPSResolverFn: func(logger model.Logger, URL string) model.Resolver {
					return &mocks.Resolver{
						MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
							return []string{}, mockedErr
						},
					}
				},
			}
			resolver := tx.newParallelDNSOverHTTPSResolver(model.DiscardLogger, "dns.google.com")
			ctx := context.Background()
			addrs, err := resolver.LookupHost(ctx, "example.com")
			if !errors.Is(err, mockedErr) {
				t.Fatal("unexpected err", err)
			}
			if len(addrs) != 0 {
				t.Fatal("expected array of size 0")
			}
		})

		t.Run("when nil", func(t *testing.T) {
			tx := &Trace{
				NewParallelDNSOverHTTPSResolverFn: nil,
			}
			resolver := tx.newParallelDNSOverHTTPSResolver(model.DiscardLogger, "https://dns.google.com")
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

	t.Run("NewDialerWithoutResolverFn works as intended", func(t *testing.T) {
		t.Run("when not nil", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			tx := &Trace{
				NewDialerWithoutResolverFn: func(dl model.DebugLogger) model.Dialer {
					return &mocks.Dialer{
						MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
							return nil, mockedErr
						},
					}
				},
			}
			dialer := tx.NewDialerWithoutResolver(model.DiscardLogger)
			ctx := context.Background()
			conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
			if !errors.Is(err, mockedErr) {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("when nil", func(t *testing.T) {
			tx := &Trace{
				NewDialerWithoutResolverFn: nil,
			}
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
	})

	t.Run("NewTLSHandshakerStdlibFn works as intended", func(t *testing.T) {
		t.Run("when not nil", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			tx := &Trace{
				NewTLSHandshakerStdlibFn: func(dl model.DebugLogger) model.TLSHandshaker {
					return &mocks.TLSHandshaker{
						MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
							return nil, tls.ConnectionState{}, mockedErr
						},
					}
				},
			}
			thx := tx.NewTLSHandshakerStdlib(model.DiscardLogger)
			ctx := context.Background()
			conn, state, err := thx.Handshake(ctx, &mocks.Conn{}, &tls.Config{})
			if !errors.Is(err, mockedErr) {
				t.Fatal("unexpected err", err)
			}
			if !reflect.ValueOf(state).IsZero() {
				t.Fatal("state is not a zero value")
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("when nil", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			tx := &Trace{
				NewTLSHandshakerStdlibFn: nil,
			}
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
			conn, state, err := thx.Handshake(ctx, tcpConn, tlsConfig)
			if !errors.Is(err, mockedErr) {
				t.Fatal("unexpected err", err)
			}
			if !reflect.ValueOf(state).IsZero() {
				t.Fatal("state is not a zero value")
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})
	})

	t.Run("TimeNowFn works as intended", func(t *testing.T) {
		fixedTime := time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
		tx := &Trace{
			TimeNowFn: func() time.Time {
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
			TimeNowFn: func() time.Time {
				return t1
			},
		}
		if tx.TimeSince(t0) != 10*time.Second {
			t.Fatal("apparently Trace.Since is broken")
		}
	})
}
