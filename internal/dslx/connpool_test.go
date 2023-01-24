package dslx

import (
	"errors"
	"io"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func mockConnWithErr(err error) io.Closer {
	return &mocks.Conn{
		MockClose: func() error {
			return err
		},
	}
}
func mockQUICConnWithErr(err error) io.Closer {
	return &quicCloserConn{
		&mocks.QUICEarlyConnection{
			MockCloseWithError: func(code quic.ApplicationErrorCode, reason string) error {
				return nil
			},
		},
	}
}

func TestMaybeTrack(t *testing.T) {
	type ConnPoolTest struct {
		name        string
		mockConn    io.Closer
		expectedLen int
	}
	tests := []ConnPoolTest{
		{
			name:        "MaybeTrack: nil",
			mockConn:    nil,
			expectedLen: 0,
		},
		{
			name:        "MaybeTrack: good conn",
			mockConn:    mockConnWithErr(nil),
			expectedLen: 1,
		},
		{
			name:        "MaybeTrack: good quic conn",
			mockConn:    mockQUICConnWithErr(nil),
			expectedLen: 1,
		},
	}
	for _, test := range tests {
		connpool := &ConnPool{}
		connpool.MaybeTrack(test.mockConn)
		if len(connpool.v) != test.expectedLen {
			t.Fatalf("%s: expected # of conns: %d, got: %d", test.name, test.expectedLen, len(connpool.v))
		}
	}
}

func TestClose(t *testing.T) {
	type testConnPool struct {
		name string
		p    ConnPool
	}
	mockErr := errors.New("mocked")

	tests := []*testConnPool{
		{
			name: "Close: all Close() succeed",
			p: ConnPool{
				v: []io.Closer{
					mockConnWithErr(nil),
					mockQUICConnWithErr(nil),
				},
			},
		},
		{
			name: "Close: 1 Close() call fails",
			p: ConnPool{
				v: []io.Closer{
					mockConnWithErr(nil),
					mockConnWithErr(mockErr),
				},
			},
		},
	}
	for _, test := range tests {
		err := test.p.Close()
		if err != nil { // Close() always returns a nil error
			t.Fatalf("%s: unexpected error %s", test.name, err)
		}
		if test.p.v != nil {
			t.Fatalf("%s: v should be reset but is not", test.name)
		}
	}
}
