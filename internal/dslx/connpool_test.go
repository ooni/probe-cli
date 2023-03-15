package dslx

import (
	"errors"
	"io"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

/*
Test cases:
- Maybe track connections
	- with nil
	- with connection
	- with quic connection
- Close ConnPool
	- all Close() calls succeed
	- one Close() call fails
*/

func closeConnWithErr(err error) io.Closer {
	return &mocks.Conn{
		MockClose: func() error {
			return err
		},
	}
}

func closeQUICConnWithErr(err error) io.Closer {
	return &quicCloserConn{
		&mocks.QUICEarlyConnection{
			MockCloseWithError: func(code quic.ApplicationErrorCode, reason string) error {
				return nil
			},
		},
	}
}

func TestConnPool(t *testing.T) {
	type connpoolTest struct {
		mockConn io.Closer
		want     int // len of connpool.v
	}
	t.Run("Maybe track connections", func(t *testing.T) {
		tests := map[string]connpoolTest{
			"with nil":             {mockConn: nil, want: 0},
			"with connection":      {mockConn: closeConnWithErr(nil), want: 1},
			"with quic connection": {mockConn: closeQUICConnWithErr(nil), want: 1},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				connpool := &ConnPool{}
				connpool.MaybeTrack(tt.mockConn)
				if len(connpool.v) != tt.want {
					t.Fatalf("expected %d tracked connections, got: %d", tt.want, len(connpool.v))
				}
			})
		}
	})
	t.Run("Close ConnPool", func(t *testing.T) {
		mockErr := errors.New("mocked")
		tests := map[string]struct {
			pool *ConnPool
		}{
			"all Close() calls succeed": {
				pool: &ConnPool{
					v: []io.Closer{
						closeConnWithErr(nil),
						closeQUICConnWithErr(nil),
					},
				},
			},
			"one Close() call fails": {
				pool: &ConnPool{
					v: []io.Closer{
						closeConnWithErr(nil),
						closeConnWithErr(mockErr),
					},
				},
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				err := tt.pool.Close()
				if err != nil { // Close() should always return nil
					t.Fatalf("unexpected error %s", err)
				}
				if tt.pool.v != nil {
					t.Fatalf("v should be reset but is not")
				}
			})
		}
	})
}
