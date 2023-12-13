package dslx

import (
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEndpoint(t *testing.T) {
	idGen := &atomic.Int64{}
	idGen.Add(42)

	t.Run("Create new endpoint", func(t *testing.T) {
		testEndpoint := NewEndpoint(
			"network",
			"10.9.8.76",
			EndpointOptionDomain("www.example.com"),
			EndpointOptionTags("antani"),
		)
		if testEndpoint.Network != "network" {
			t.Fatalf("unexpected network")
		}
		if testEndpoint.Address != "10.9.8.76" {
			t.Fatalf("unexpected address")
		}
		if testEndpoint.Domain != "www.example.com" {
			t.Fatalf("unexpected domain")
		}
		if diff := cmp.Diff([]string{"antani"}, testEndpoint.Tags); diff != "" {
			t.Fatal(diff)
		}
	})
}
