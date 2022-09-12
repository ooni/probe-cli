package measurexlite

import (
	"context"
	"errors"
	"math"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNewDialerWithoutResolver(t *testing.T) {
	t.Run("NewDialerWithoutResolver creates a wrapped dialer", func(t *testing.T) {
		underlying := &mocks.Dialer{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NewDialerWithoutResolverFn = func(dl model.DebugLogger) model.Dialer {
			return underlying
		}
		dialer := trace.NewDialerWithoutResolver(model.DiscardLogger)
		dt := dialer.(*dialerTrace)
		if dt.d != underlying {
			t.Fatal("invalid dialer")
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
		underlying := &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				gotTrace := netxlite.ContextTraceOrDefault(ctx)
				hasCorrectTrace = (gotTrace == trace)
				return nil, expectedErr
			},
		}
		trace.NewDialerWithoutResolverFn = func(dl model.DebugLogger) model.Dialer {
			return underlying
		}
		dialer := trace.NewDialerWithoutResolver(model.DiscardLogger)
		ctx := context.Background()
		conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
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
		underlying := &mocks.Dialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		trace.NewDialerWithoutResolverFn = func(dl model.DebugLogger) model.Dialer {
			return underlying
		}
		dialer := trace.NewDialerWithoutResolver(model.DiscardLogger)
		dialer.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("DialContext saves into the trace", func(t *testing.T) {
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = td.Now // deterministic time tracking
		dialer := trace.NewDialerWithoutResolver(model.DiscardLogger)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // we cancel immediately so connect is ~instantaneous
		conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
		if err == nil || err.Error() != netxlite.FailureInterrupted {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}

		expectedFailure := netxlite.FailureInterrupted

		t.Run("for TCPConnect", func(t *testing.T) {
			events := trace.TCPConnects()
			if len(events) != 1 {
				t.Fatal("expected to see single TCPConnect event")
			}
			expect := &model.ArchivalTCPConnectResult{
				IP:   "1.1.1.1",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Blocked: nil,
					Failure: &expectedFailure,
					Success: false,
				},
				T: time.Second.Seconds(),
			}
			got := events[0]
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("for NetworkEvents", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 1 {
				t.Fatal("expected to see single NetworkEvent event")
			}
			expectedFailure := netxlite.FailureInterrupted
			expect := &model.ArchivalNetworkEvent{
				Address:       "1.1.1.1:443",
				Failure:       &expectedFailure,
				NumBytes:      0,
				Operation:     netxlite.ConnectOperation,
				Proto:         "tcp",
				T0:            0,
				T:             time.Second.Seconds(),
				TransactionID: 0,
				Tags:          []string{},
			}
			got := events[0]
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("DialContext discards events when buffer is full", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.tcpConnect = make(chan *model.ArchivalTCPConnectResult) // no buffer
		trace.networkEvent = make(chan *model.ArchivalNetworkEvent)   // ditto
		dialer := trace.NewDialerWithoutResolver(model.DiscardLogger)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // we cancel immediately so connect is ~instantaneous
		conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
		if err == nil || err.Error() != netxlite.FailureInterrupted {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}

		t.Run("for TCPConnect", func(t *testing.T) {
			events := trace.TCPConnects()
			if len(events) != 0 {
				t.Fatal("expected to see no TCPConnect events")
			}
		})

		t.Run("for NetworkEvents", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 0 {
				t.Fatal("expected to see no NetworkEvent events")
			}
		})
	})

	t.Run("DialContext ignores UDP connect attempts", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		dialer := trace.NewDialerWithoutResolver(model.DiscardLogger)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // we cancel immediately so connect is ~instantaneous
		conn, err := dialer.DialContext(ctx, "udp", "1.1.1.1:443")
		if err == nil || err.Error() != netxlite.FailureInterrupted {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}

		t.Run("for TCP connect", func(t *testing.T) {
			events := trace.TCPConnects()
			if len(events) != 0 {
				t.Fatal("expected to see no TCPConnect events")
			}
		})

		t.Run("for NetworkEvents", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 0 {
				t.Fatal("expected to see no NetworkEvent events")
			}
		})
	})

	t.Run("DialContext uses a dialer without a resolver", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		dialer := trace.NewDialerWithoutResolver(model.DiscardLogger)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()                                                      // we cancel immediately so connect is ~instantaneous
		conn, err := dialer.DialContext(ctx, "udp", "dns.google:443") // domain
		if !errors.Is(err, netxlite.ErrNoResolver) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		events := trace.TCPConnects()
		if len(events) != 0 {
			t.Fatal("expected to see no TCPConnect events")
		}
	})
}

func TestFirstTCPConnect(t *testing.T) {
	t.Run("returns nil when buffer is empty", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		got := trace.FirstTCPConnectOrNil()
		if got != nil {
			t.Fatal("expected nil event")
		}
	})

	t.Run("return first non-nil TCPConnect", func(t *testing.T) {
		filler := func(tx *Trace, events []*model.ArchivalTCPConnectResult) {
			for _, ev := range events {
				tx.tcpConnect <- ev
			}
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		expect := []*model.ArchivalTCPConnectResult{{
			IP:   "1.1.1.1",
			Port: 443,
			Status: model.ArchivalTCPConnectStatus{
				Blocked: nil,
				Failure: nil,
				Success: true,
			},
		}, {
			IP:   "0.0.0.0",
			Port: 443,
			Status: model.ArchivalTCPConnectStatus{
				Blocked: nil,
				Failure: nil,
				Success: true,
			},
		}}
		filler(trace, expect)
		got := trace.FirstTCPConnectOrNil()
		if diff := cmp.Diff(got, expect[0]); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestArchivalSplitHostPort(t *testing.T) {
	addr, port := archivalSplitHostPort("1.1.1.1") // missing port
	if addr != "" {
		t.Fatal("invalid addr", addr)
	}
	if port != "" {
		t.Fatal("invalid port", port)
	}
}

func TestArchivalPortToString(t *testing.T) {
	t.Run("with invalid number", func(t *testing.T) {
		port := archivalPortToString("antani")
		if port != 0 {
			t.Fatal("invalid port")
		}
	})

	t.Run("with negative number", func(t *testing.T) {
		port := archivalPortToString("-1")
		if port != 0 {
			t.Fatal("invalid port")
		}
	})

	t.Run("with too-large positive number", func(t *testing.T) {
		port := archivalPortToString(strconv.Itoa(math.MaxUint16 + 1))
		if port != 0 {
			t.Fatal("invalid port")
		}
	})
}
