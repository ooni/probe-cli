package tlsmiddlebox

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/context"
)

func TestMeasureTLS(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	sni := "example.com"
	ctx := context.Background()
	tlsEvents := make(chan *CompleteTrace, 1)
	m.MeasureTLS(ctx, "1.1.1.1:443", sni, tlsEvents)
	out := GetTLSEvents(tlsEvents)
	if len(out) != 1 {
		t.Fatal("expected no. of traces: 2, got", len(out))
	}
}

func TestMeasureTLS_with_full_buffer(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	sni := "example.com"
	ctx := context.Background()
	tlsEvents := make(chan *CompleteTrace) // channel without buffer
	m.MeasureTLS(ctx, "1.1.1.1:443", sni, tlsEvents)
	out := GetTLSEvents(tlsEvents)
	if len(out) != 0 {
		t.Fatal("expected 0 traces, found", len(out))
	}
}

func TestIterativeTrace(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	sni := "example.com"
	ctx := context.Background()
	trace := m.IterativeTrace(ctx, "1.1.1.1:443", sni)
	if len(trace.Iterations) <= 0 {
		t.Fatal("no iterations recorded")
	}
	if trace.Address != "1.1.1.1:443" {
		t.Fatal("address does not match")
	}
	if trace.SNI != "example.com" {
		t.Fatal("invalid SNI")
	}
	for i, ev := range trace.Iterations {
		if ev.TTL != i+1 {
			t.Fatal("invalid iteration alignment")
		}
	}
}

func TestIterativeTrace_channels(t *testing.T) {
	runHelper := func(iterations int, buffer int) []*IterEvent {
		m := NewExperimentMeasurer(Config{})
		ctx := context.Background()
		sni := "example.com"
		events := make(chan *IterEvent, buffer)
		m.iterativeTrace(ctx, "1.1.1.1:443", sni, iterations, events)
		return GetTraceEvents(events)
	}
	t.Run("with buffered channel", func(t *testing.T) {
		iterations := 10
		out := runHelper(iterations, 10)
		if len(out) != iterations {
			t.Fatal("invalid number of iterations")
		}
	})
	t.Run("with full channel", func(t *testing.T) {
		iterations := 10
		out := runHelper(iterations, 0)
		if len(out) != 0 {
			t.Fatal("invalid number of iterations")
		}
	})
}

func TestHandshakeWithTTL(t *testing.T) {
	ctx := context.Background()
	sni := "example.com"
	expectedFailure := "generic_timeout_error"
	expected := IterEvent{
		Failure: &expectedFailure,
		TTL:     1,
	}
	out := HandshakeWithTTL(ctx, "1.1.1.1:443", sni, 1) // set TTL to 1
	if out.Failure == nil {
		t.Fatal("expected error")
	}
	if *(out.Failure) != *(expected.Failure) {
		t.Fatal("unexpected error:", *(out.Failure))
	}
	if out.TTL != expected.TTL {
		t.Fatal("wrong TTL recorded, found:", out.TTL)
	}
}

func TestAlignIterEvents(t *testing.T) {
	var (
		timeoutFailure = "generic_timeout_err"
		resetFailure   = "connection_reset"
	)
	tests := []struct {
		name  string
		input []*IterEvent
		want  []*IterEvent
	}{{
		name: "with failure",
		input: []*IterEvent{{
			Failure: &timeoutFailure,
			TTL:     2,
		}, {
			Failure: &timeoutFailure,
			TTL:     3,
		}, {
			Failure: &timeoutFailure,
			TTL:     1,
		}},
		want: []*IterEvent{{
			Failure: &timeoutFailure,
			TTL:     1,
		}, {
			Failure: &timeoutFailure,
			TTL:     2,
		}, {
			Failure: &timeoutFailure,
			TTL:     3,
		}},
	}, {
		name: "without failure",
		input: []*IterEvent{{
			Failure: nil,
			TTL:     2,
		}, {
			Failure: nil,
			TTL:     3,
		}, {
			Failure: &timeoutFailure,
			TTL:     1,
		}},
		want: []*IterEvent{{
			Failure: &timeoutFailure,
			TTL:     1,
		}, {
			Failure: nil,
			TTL:     2,
		}},
	}, {
		name: "with connection reset",
		input: []*IterEvent{{
			Failure: &resetFailure,
			TTL:     2,
		}, {
			Failure: &resetFailure,
			TTL:     3,
		}, {
			Failure: &timeoutFailure,
			TTL:     1,
		}},
		want: []*IterEvent{{
			Failure: &timeoutFailure,
			TTL:     1,
		}, {
			Failure: &resetFailure,
			TTL:     2,
		}},
	}}

	for _, tt := range tests {
		out := alignIterEvents(tt.input)
		if diff := cmp.Diff(out, tt.want); diff != "" {
			t.Fatal(diff)
		}
	}
}
