package dslx

import (
	"errors"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/quic-go/quic-go"
)

/*
Test cases:
- Maybe track connections:
	- with nil
	- with connection
	- with quic connection

- Close MinimalRuntime:
	- all Close() calls succeed
	- one Close() call fails
*/

func closeableConnWithErr(err error) io.Closer {
	return &mocks.Conn{
		MockClose: func() error {
			return err
		},
	}
}

func closeableQUICConnWithErr(err error) io.Closer {
	return &quicCloserConn{
		&mocks.QUICEarlyConnection{
			MockCloseWithError: func(code quic.ApplicationErrorCode, reason string) error {
				return err
			},
		},
	}
}

func TestMinimalRuntime(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		mockConn io.Closer
		want     int // len of (*minimalRuntime).v
	}

	t.Run("Maybe track connections", func(t *testing.T) {
		tests := map[string]testcase{
			"with nil":             {mockConn: nil, want: 0},
			"with connection":      {mockConn: closeableConnWithErr(nil), want: 1},
			"with quic connection": {mockConn: closeableQUICConnWithErr(nil), want: 1},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				rt := NewMinimalRuntime()
				rt.MaybeTrackConn(tt.mockConn)
				if len(rt.v) != tt.want {
					t.Fatalf("expected %d tracked connections, got: %d", tt.want, len(rt.v))
				}
			})
		}
	})

	t.Run("Close MinimalRuntime", func(t *testing.T) {
		mockErr := errors.New("mocked")
		tests := map[string]struct {
			rt *MinimalRuntime
		}{
			"all Close() calls succeed": {
				rt: &MinimalRuntime{
					v: []io.Closer{
						closeableConnWithErr(nil),
						closeableQUICConnWithErr(nil),
					},
				},
			},
			"one Close() call fails": {
				rt: &MinimalRuntime{
					v: []io.Closer{
						closeableConnWithErr(nil),
						closeableConnWithErr(mockErr),
					},
				},
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				err := tt.rt.Close()
				if err != nil { // Close() should always return nil
					t.Fatalf("unexpected error %s", err)
				}
				if tt.rt.v != nil {
					t.Fatalf("v should be reset but is not")
				}
			})
		}
	})
}
