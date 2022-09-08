package tlsmiddlebox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
)

func TestIterativeTrace(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer server.Close()
		URL, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		m := NewExperimentMeasurer(Config{})
		zeroTime := time.Now()
		ctx := context.Background()
		trace := m.startIterativeTrace(ctx, 0, zeroTime, model.DiscardLogger, URL.Host, "example.com")
		if trace.SNI != "example.com" {
			t.Fatal("unexpected servername")
		}
		if len(trace.Iterations) != 1 {
			t.Fatal("unexpected number of iterations")
		}
		for i, ev := range trace.Iterations {
			if ev.TTL != i+1 {
				t.Fatal("unexpected TTL value")
			}
		}
	})

	t.Run("failure case", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}
		server := filtering.NewTLSServer(filtering.TLSActionTimeout)
		defer server.Close()
		th := "tlshandshake://" + server.Endpoint()
		URL, err := url.Parse(th)
		if err != nil {
			t.Fatal(err)
		}
		URL.Scheme = "tlshandshake"
		m := NewExperimentMeasurer(Config{})
		zeroTime := time.Now()
		ctx := context.Background()
		trace := m.startIterativeTrace(ctx, 0, zeroTime, model.DiscardLogger, URL.Host, "example.com")
		if trace.SNI != "example.com" {
			t.Fatal("unexpected servername")
		}
		if len(trace.Iterations) != 20 {
			t.Fatal("unexpected number of iterations")
		}
		for i, ev := range trace.Iterations {
			if ev.TTL != i+1 {
				t.Fatal("unexpected TTL value")
			}
		}
	})
}

func TestHandshakeWithTTL(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer server.Close()
		URL, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		m := NewExperimentMeasurer(Config{})
		tr := &IterativeTrace{}
		zeroTime := time.Now()
		ctx := context.Background()
		wg := new(sync.WaitGroup)
		wg.Add(1)
		m.handshakeWithTTL(ctx, 0, zeroTime, model.DiscardLogger, URL.Host, "example.com", 3, tr, wg)
		if len(tr.Iterations) != 1 {
			t.Fatal("unexpected number of iterations")
		}
		iter := tr.Iterations[0]
		if iter.TTL != 3 {
			t.Fatal("unexpected TTL value")
		}
		if iter.Handshake == nil || iter.Handshake.ServerName != "example.com" {
			t.Fatal("unexpected servername")
		}
		if iter.Handshake.Failure != nil {
			t.Fatal("unexpected error", *iter.Handshake.Failure)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		server := filtering.NewTLSServer(filtering.TLSActionReset)
		defer server.Close()
		th := "tlshandshake://" + server.Endpoint()
		URL, err := url.Parse(th)
		if err != nil {
			t.Fatal(err)
		}
		URL.Scheme = "tlshandshake"
		m := NewExperimentMeasurer(Config{})
		tr := &IterativeTrace{}
		zeroTime := time.Now()
		ctx := context.Background()
		wg := new(sync.WaitGroup)
		wg.Add(1)
		m.handshakeWithTTL(ctx, 0, zeroTime, model.DiscardLogger, URL.Host, "example.com", 3, tr, wg)
		if len(tr.Iterations) != 1 {
			t.Fatal("unexpected number of iterations")
		}
		iter := tr.Iterations[0]
		if iter.TTL != 3 {
			t.Fatal("unexpected TTL value")
		}
		if iter.Handshake == nil || iter.Handshake.ServerName != "example.com" {
			t.Fatal("unexpected servername")
		}
		if *iter.Handshake.Failure != netxlite.FailureConnectionReset {
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
