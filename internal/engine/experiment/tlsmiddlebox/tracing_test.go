package tlsmiddlebox

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestIterativeTrace(t *testing.T) {
	zeroTime := time.Now()
	m := NewExperimentMeasurer(Config{})
	ctx := context.Background()
	trace := m.IterativeTrace(ctx, 0, zeroTime, model.DiscardLogger, "1.1.1.1:443", "example.com")
	if trace.SNI != "example.com" {
		t.Fatal("unexpected servername")
	}
	if len(trace.Iterations) <= 0 {
		t.Fatal("no iterations recorded")
	}
	for i, ev := range trace.Iterations {
		if ev.TTL != i+1 {
			t.Fatal("unexpected TTL value")
		}
	}
}

func TestHandshakeWithTTL(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
		m := NewExperimentMeasurer(Config{})
		tr := &IterativeTrace{}
		zeroTime := time.Now()
		ctx := context.Background()
		wg := new(sync.WaitGroup)
		wg.Add(1)
		m.handshakeWithTTL(ctx, 0, zeroTime, model.DiscardLogger, "1.1.1.1:443", "example.com", 1, tr, wg)
		if len(tr.Iterations) != 1 {
			t.Fatal("unexpected number of iterations")
		}
		iter := tr.Iterations[0]
		if iter.TTL != 1 {
			t.Fatal("unexpected TTL value")
		}
		if iter.Handshake.Failure == nil || *iter.Handshake.Failure != netxlite.FailureGenericTimeoutError {
			t.Fatal("unexpected error", *iter.Handshake.Failure)
		}
	})

	t.Run("on connect failure", func(t *testing.T) {
		m := NewExperimentMeasurer(Config{})
		tr := &IterativeTrace{}
		zeroTime := time.Now()
		ctx := context.Background()
		wg := new(sync.WaitGroup)
		wg.Add(1)
		m.handshakeWithTTL(ctx, 0, zeroTime, model.DiscardLogger, "1.1.1.1.1:443", "example.com", 1, tr, wg)
		if len(tr.Iterations) != 1 {
			t.Fatal("unexpected number of iterations")
		}
		iter := tr.Iterations[0]
		if iter.TTL != 1 {
			t.Fatal("unexpected TTL value")
		}
		if iter.Handshake.Failure == nil || *iter.Handshake.Failure != netxlite.FailureDNSNXDOMAINError {
			t.Fatal("unexpected error", *iter.Handshake.Failure)
		}
	})
}

func TestAlignIterations(t *testing.T) {
	var (
		failureTimeout         = "generic_timeout_err"
		failureConnectionReset = "connection_reset"
	)
	tests := []struct {
		name  string
		input []*Iteration
		want  []*Iteration
	}{{
		name: "with failure",
		input: []*Iteration{{
			TTL: 2,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}, {
			TTL: 3,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}, {
			TTL: 1,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}},
		want: []*Iteration{{
			TTL: 1,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}, {
			TTL: 2,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}, {
			TTL: 3,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}},
	}, {
		name: "without failure",
		input: []*Iteration{{
			TTL: 2,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: nil,
			},
		}, {
			TTL: 3,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}, {
			TTL: 1,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}},
		want: []*Iteration{{
			TTL: 1,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}, {
			TTL: 2,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: nil,
			},
		}},
	}, {
		name: "with connection reset",
		input: []*Iteration{{
			TTL: 2,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureConnectionReset,
			},
		}, {
			TTL: 3,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureConnectionReset,
			},
		}, {
			TTL: 1,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}},
		want: []*Iteration{{
			TTL: 1,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureTimeout,
			},
		}, {
			TTL: 2,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: &failureConnectionReset,
			},
		}},
	}, {
		name:  "empty input",
		input: []*Iteration{},
		want:  []*Iteration{},
	}}

	for _, tt := range tests {
		out := alignIterations(tt.input)
		if diff := cmp.Diff(out, tt.want); diff != "" {
			t.Fatal(diff)
		}
	}
}
