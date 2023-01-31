package dslx

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

type tcpTest struct {
	expectedConn net.Conn
	expectedErr  error
	wasClosed    bool
	name         string
	dialer       *mocks.Dialer
}

var wasClosed bool = false

func TestApplyTCP(t *testing.T) {
	plainConn := &mocks.Conn{
		MockClose: func() error {
			wasClosed = true
			return nil
		},
	}
	tests := []tcpTest{
		{
			name:         "with EOF",
			expectedConn: nil,
			expectedErr:  io.EOF,
			dialer: &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
					return nil, io.EOF
				},
			},
		},
		{
			name:         "success",
			expectedConn: plainConn,
			expectedErr:  nil,
			wasClosed:    true,
			dialer: &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
					return plainConn, nil
				},
			},
		},
	}

	for _, test := range tests {
		pool := &ConnPool{}
		tcpConnect := &tcpConnectFunc{pool, test.dialer}
		endpoint := &Endpoint{
			Address:     "1.2.3.4:567",
			Network:     "tcp",
			IDGenerator: &atomic.Int64{},
			Logger:      model.DiscardLogger,
			ZeroTime:    time.Time{},
		}
		res := tcpConnect.Apply(context.Background(), endpoint)
		if res.Error != test.expectedErr {
			t.Fatalf("%s: expected error %s, got %s", test.name, test.expectedErr, res.Error)
		}
		if res.State.Conn != test.expectedConn {
			t.Fatalf("%s: expected conn %s, got %s", test.name, test.expectedConn, res.State.Conn)
		}
		pool.Close()
		if wasClosed != test.wasClosed {
			t.Fatalf("%s: expected closeErr %v, got %v", test.name, test.wasClosed, wasClosed)
		}
		wasClosed = false
	}
}
