package dslx

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestTCPConnect(t *testing.T) {
	t.Run("Get tcpConnectFunc", func(t *testing.T) {
		f := TCPConnect(
			&ConnPool{},
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
			dialer     model.Dialer
			expectConn net.Conn
			expectErr  error
			closed     bool
		}{
			"with EOF": {expectConn: nil, expectErr: io.EOF, closed: false, dialer: eofDialer},
			"success":  {expectConn: plainConn, expectErr: nil, closed: true, dialer: goodDialer},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				pool := &ConnPool{}
				tcpConnect := &tcpConnectFunc{pool, tt.dialer}
				endpoint := &Endpoint{
					Address:     "1.2.3.4:567",
					Network:     "tcp",
					IDGenerator: &atomic.Int64{},
					Logger:      model.DiscardLogger,
					ZeroTime:    time.Time{},
				}
				res := tcpConnect.Apply(context.Background(), endpoint)
				if res.Error != tt.expectErr {
					t.Fatalf("unexpected error: %s", res.Error)
				}
				if res.State.Conn != tt.expectConn {
					t.Fatalf("unexpected conn: %s", res.State.Conn)
				}
				pool.Close()
				if wasClosed != tt.closed {
					t.Fatalf("unexpected connection closed state: %v", wasClosed)
				}
			})
			wasClosed = false
		}
	})
}
