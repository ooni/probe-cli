package dslx

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestEndpoint(t *testing.T) {
	idGen := &atomic.Int64{}
	idGen.Add(42)
	zt := time.Now()

	t.Run("Create new endpoint", func(t *testing.T) {
		testEndpoint := NewEndpoint(
			"network",
			"a.b.c.d",
			EndpointOptionDomain("www.example.com"),
			EndpointOptionIDGenerator(idGen),
			EndpointOptionLogger(model.DiscardLogger),
			EndpointOptionZeroTime(zt),
		)
		if testEndpoint.Network != "network" {
			t.Fatalf("unexpected network")
		}
		if testEndpoint.Address != "a.b.c.d" {
			t.Fatalf("unexpected address")
		}
		if testEndpoint.Domain != "www.example.com" {
			t.Fatalf("unexpected domain")
		}
		if testEndpoint.IDGenerator != idGen {
			t.Fatalf("unexpected IDGenerator")
		}
		if testEndpoint.Logger != model.DiscardLogger {
			t.Fatalf("unexpected logger")
		}
		if testEndpoint.ZeroTime != zt {
			t.Fatalf("unexpected zero time")
		}
	})
}
