package dslx

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestTCPConnect(t *testing.T) {
	t.Run("Get tcpConnectFunc", func(t *testing.T) {
		f := TCPConnect(
			NewMinimalRuntime(),
		)
		if _, ok := f.(*tcpConnectFunc); !ok {
			t.Fatal("unexpected type. Expected: tcpConnectFunc")
		}
	})

	t.Run("Apply tcpConnectFunc", func(t *testing.T) {
		wasClosed := false
		plainConn := &mocks.Conn{
			MockClose: func() error {
				wasClosed = true
				return nil
			},
		}
		eofDialer := &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, io.EOF
			},
		}

		goodDialer := &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return plainConn, nil
			},
		}

		tests := map[string]struct {
			tags       []string
			dialer     model.Dialer
			expectConn net.Conn
			expectErr  error
			closed     bool
		}{
			"with EOF": {
				tags:       []string{},
				expectConn: nil,
				expectErr:  io.EOF,
				closed:     false,
				dialer:     eofDialer,
			},
			"success": {
				tags:       []string{"antani"},
				expectConn: plainConn,
				expectErr:  nil,
				closed:     true,
				dialer:     goodDialer,
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				rt := NewMinimalRuntime()
				tcpConnect := &tcpConnectFunc{tt.dialer, rt}
				endpoint := &Endpoint{
					Address:     "1.2.3.4:567",
					Network:     "tcp",
					IDGenerator: &atomic.Int64{},
					Logger:      model.DiscardLogger,
					Tags:        tt.tags,
					ZeroTime:    time.Time{},
				}
				res := tcpConnect.Apply(context.Background(), endpoint)
				if res.Error != tt.expectErr {
					t.Fatalf("unexpected error: %s", res.Error)
				}
				if res.State == nil || res.State.Conn != tt.expectConn {
					t.Fatal("unexpected conn")
				}
				rt.Close()
				if wasClosed != tt.closed {
					t.Fatalf("unexpected connection closed state: %v", wasClosed)
				}
				if len(tt.tags) > 0 {
					if res.State == nil {
						t.Fatal("expected non-nil res.State")
					}
					if diff := cmp.Diff([]string{"antani"}, res.State.Trace.Tags()); diff != "" {
						t.Fatal(diff)
					}
				}
			})
			wasClosed = false
		}
	})
}

// Make sure we get a valid dialer if no mocked dialer is configured
func TestDialerOrDefault(t *testing.T) {
	f := &tcpConnectFunc{
		rt:     NewMinimalRuntime(),
		dialer: nil,
	}
	dialer := f.dialerOrDefault(measurexlite.NewTrace(0, time.Now()), model.DiscardLogger)
	if dialer == nil {
		t.Fatal("expected non-nil dialer here")
	}
}
