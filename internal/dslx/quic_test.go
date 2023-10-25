package dslx

import (
	"context"
	"crypto/tls"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/quic-go/quic-go"
)

/*
Test cases:
- Apply quicHandshakeFunc:
  - with EOF
  - success
  - with sni
*/
func TestQUICHandshake(t *testing.T) {
	t.Run("Apply quicHandshakeFunc", func(t *testing.T) {
		wasClosed := false
		plainConn := &mocks.QUICEarlyConnection{
			MockCloseWithError: func(code quic.ApplicationErrorCode, reason string) error {
				wasClosed = true
				return nil
			},
			MockConnectionState: func() quic.ConnectionState {
				return quic.ConnectionState{}
			},
		}

		eofDialer := &mocks.QUICDialer{
			MockDialContext: func(ctx context.Context, address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlyConnection, error) {
				return nil, io.EOF
			},
		}

		goodDialer := &mocks.QUICDialer{
			MockDialContext: func(ctx context.Context, address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlyConnection, error) {
				return plainConn, nil
			},
		}

		tests := map[string]struct {
			dialer     model.QUICDialer
			sni        string
			tags       []string
			expectConn quic.EarlyConnection
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
			"with sni": {
				tags:       []string{},
				expectConn: plainConn,
				expectErr:  nil,
				closed:     true,
				dialer:     goodDialer,
				sni:        "sni.com",
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				rt := NewRuntimeMeasurexLite(model.DiscardLogger, time.Now(), RuntimeMeasurexLiteOptionMeasuringNetwork(&mocks.MeasuringNetwork{
					MockNewQUICDialerWithoutResolver: func(listener model.UDPListener, logger model.DebugLogger, w ...model.QUICDialerWrapper) model.QUICDialer {
						return tt.dialer
					},
				}))
				quicHandshake := QUICHandshake(rt, TLSHandshakeOptionServerName(tt.sni))
				endpoint := &Endpoint{
					Address: "1.2.3.4:567",
					Network: "udp",
					Tags:    tt.tags,
				}
				res := quicHandshake.Apply(context.Background(), NewMaybeWithValue(endpoint))
				if res.Error != tt.expectErr {
					t.Fatalf("unexpected error: %s", res.Error)
				}
				if res.Error == nil && res.State.QUICConn != tt.expectConn {
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
