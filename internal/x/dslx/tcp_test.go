package dslx

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestTCPConnect(t *testing.T) {
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
				rt := NewRuntimeMeasurexLite(model.DiscardLogger, time.Now(), RuntimeMeasurexLiteOptionMeasuringNetwork(&mocks.MeasuringNetwork{
					MockNewDialerWithoutResolver: func(dl model.DebugLogger, w ...model.DialerWrapper) model.Dialer {
						return tt.dialer
					},
				}))
				tcpConnect := TCPConnect(rt)
				endpoint := &Endpoint{
					Address: "1.2.3.4:567",
					Network: "tcp",
					Tags:    tt.tags,
				}
				res := tcpConnect.Apply(context.Background(), NewMaybeWithValue(endpoint))
				if res.Error != tt.expectErr {
					t.Fatalf("unexpected error: %s", res.Error)
				}
				if res.Error == nil && res.State.Conn != tt.expectConn {
					t.Fatalf("unexpected conn %v", res.State)
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
