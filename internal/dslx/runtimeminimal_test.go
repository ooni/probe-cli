package dslx

import (
	"errors"
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
				rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
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

	t.Run("IDGenerator", func(t *testing.T) {
		rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
		out := rt.IDGenerator()
		if out == nil {
			t.Fatal("expected non-nil pointer")
		}
	})

	t.Run("Logger", func(t *testing.T) {
		rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
		out := rt.Logger()
		if out == nil {
			t.Fatal("expected non-nil pointer")
		}
	})

	t.Run("ZeroTime", func(t *testing.T) {
		rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
		out := rt.ZeroTime()
		if out.IsZero() {
			t.Fatal("expected non-zero time")
		}
	})

	t.Run("Trace", func(t *testing.T) {
		tags := []string{"antani", "mascetti", "melandri"}
		rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
		now := time.Now()
		trace := rt.NewTrace(10, now, tags...)

		t.Run("CloneBytesReceivedMap", func(t *testing.T) {
			out := trace.CloneBytesReceivedMap()
			if out == nil || len(out) != 0 {
				t.Fatal("expected zero-length map")
			}
		})

		t.Run("DNSLookupsFromRoundTrip", func(t *testing.T) {
			out := trace.DNSLookupsFromRoundTrip()
			if out == nil || len(out) != 0 {
				t.Fatal("expected zero-length slice")
			}
		})

		t.Run("Index", func(t *testing.T) {
			out := trace.Index()
			if out != 10 {
				t.Fatal("expected 10, got", out)
			}
		})

		t.Run("NetworkEvents", func(t *testing.T) {
			out := trace.NetworkEvents()
			if out == nil || len(out) != 0 {
				t.Fatal("expected zero-length slice")
			}
		})

		t.Run("NewDialerWithoutResolver", func(t *testing.T) {
			out := trace.NewDialerWithoutResolver(model.DiscardLogger)
			if out == nil {
				t.Fatal("expected non-nil pointer")
			}
		})

		t.Run("NewParallelUDPResolver", func(t *testing.T) {
			out := trace.NewParallelUDPResolver(model.DiscardLogger, &mocks.Dialer{}, "8.8.8.8:53")
			if out == nil {
				t.Fatal("expected non-nil pointer")
			}
		})

		t.Run("NewQUICDialerWithoutResolver", func(t *testing.T) {
			out := trace.NewQUICDialerWithoutResolver(&mocks.UDPListener{}, model.DiscardLogger)
			if out == nil {
				t.Fatal("expected non-nil pointer")
			}
		})

		t.Run("NewStdlibResolver", func(t *testing.T) {
			out := trace.NewStdlibResolver(model.DiscardLogger)
			if out == nil {
				t.Fatal("expected non-nil pointer")
			}
		})

		t.Run("NewTLSHandshakerStdlib", func(t *testing.T) {
			out := trace.NewTLSHandshakerStdlib(model.DiscardLogger)
			if out == nil {
				t.Fatal("expected non-nil pointer")
			}
		})

		t.Run("QUICHandshakes", func(t *testing.T) {
			out := trace.QUICHandshakes()
			if out == nil || len(out) != 0 {
				t.Fatal("expected zero-length slice")
			}
		})

		t.Run("TCPConnects", func(t *testing.T) {
			out := trace.TCPConnects()
			if out == nil || len(out) != 0 {
				t.Fatal("expected zero-length slice")
			}
		})

		t.Run("TLSHandshakes", func(t *testing.T) {
			out := trace.TLSHandshakes()
			if out == nil || len(out) != 0 {
				t.Fatal("expected zero-length slice")
			}
		})

		t.Run("Tags", func(t *testing.T) {
			out := trace.Tags()
			if diff := cmp.Diff(tags, out); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("TimeSince", func(t *testing.T) {
			out := trace.TimeSince(now.Add(-10 * time.Second))
			if out == 0 {
				t.Fatal("expected non-zero time")
			}
		})

		t.Run("ZeroTime", func(t *testing.T) {
			out := trace.ZeroTime()
			if out.IsZero() {
				t.Fatal("expected non-zero time")
			}
		})
	})
}
