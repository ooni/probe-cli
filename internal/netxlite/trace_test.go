package netxlite

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestContextTraceOrDefault(t *testing.T) {
	t.Run("without a configured trace we get a default", func(t *testing.T) {
		ctx := context.Background()
		tx := ContextTraceOrDefault(ctx)
		_ = tx.(*traceDefault) // panic if cannot cast
	})

	t.Run("with a configured trace we get the expected trace", func(t *testing.T) {
		realTrace := &mocks.Trace{}
		ctx := ContextWithTrace(context.Background(), realTrace)
		tx := ContextTraceOrDefault(ctx)
		if tx != realTrace {
			t.Fatal("not the trace we expected")
		}
	})
}

func TestContextWithTrace(t *testing.T) {
	t.Run("panics if passed a nil trace", func(t *testing.T) {
		var called bool
		func() {
			defer func() {
				called = (recover() != nil)
			}()
			_ = ContextWithTrace(context.Background(), nil)
		}()
		if !called {
			t.Fatal("not called")
		}
	})
}
