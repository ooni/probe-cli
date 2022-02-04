package bytecounter

import (
	"context"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestSessionByteCounter(t *testing.T) {
	counter := New()
	ctx := context.Background()
	ctx = WithSessionByteCounter(ctx, counter)
	outer := ContextSessionByteCounter(ctx)
	if outer != counter {
		t.Fatal("unexpected result")
	}
}

func TestExperimentByteCounter(t *testing.T) {
	counter := New()
	ctx := context.Background()
	ctx = WithExperimentByteCounter(ctx, counter)
	outer := ContextExperimentByteCounter(ctx)
	if outer != counter {
		t.Fatal("unexpected result")
	}
}

func TestWrapWithContextByteCounters(t *testing.T) {
	var conn net.Conn = &mocks.Conn{
		MockRead: func(b []byte) (int, error) {
			return len(b), nil
		},
		MockWrite: func(b []byte) (int, error) {
			return len(b), nil
		},
	}
	sessCounter := New()
	expCounter := New()
	ctx := context.Background()
	ctx = WithSessionByteCounter(ctx, sessCounter)
	ctx = WithExperimentByteCounter(ctx, expCounter)
	conn = WrapWithContextByteCounters(ctx, conn)
	buf := make([]byte, 128)
	conn.Read(buf)
	conn.Write(buf)
	if sessCounter.Received.Load() != 128 {
		t.Fatal("invalid value")
	}
	if sessCounter.Sent.Load() != 128 {
		t.Fatal("invalid value")
	}
	if expCounter.Received.Load() != 128 {
		t.Fatal("invalid value")
	}
	if expCounter.Sent.Load() != 128 {
		t.Fatal("invalid value")
	}
}
