package dslx

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

var wantDomain string = "www.google.com"
var wantIDGenerator *atomic.Int64 = &atomic.Int64{}
var wantLogger model.Logger = model.DiscardLogger
var wantZeroTime = time.Time{}

var options []EndpointOption = []EndpointOption{
	EndpointOptionDomain(wantDomain),
	EndpointOptionIDGenerator(wantIDGenerator),
	EndpointOptionLogger(wantLogger),
	EndpointOptionZeroTime(wantZeroTime),
}

func TestNewEndpoint(t *testing.T) {
	testEndpoint := NewEndpoint("network", "a.b.c.d", options...)

	if testEndpoint.Network != "network" {
		t.Fatalf("expected: %s, got: %s", "network", testEndpoint.Network)
	}
	if testEndpoint.Address != "a.b.c.d" {
		t.Fatalf("expected: %s, got: %s", "a.b.c.d", testEndpoint.Address)
	}
	if testEndpoint.Domain != wantDomain {
		t.Fatalf("expected: %s, got: %s", wantDomain, testEndpoint.Domain)
	}
	if testEndpoint.IDGenerator != wantIDGenerator {
		t.Fatalf("expected: %v, got: %v", wantIDGenerator, testEndpoint.IDGenerator)
	}
	if testEndpoint.Logger != wantLogger {
		t.Fatalf("expected: %v, got: %v", wantLogger, testEndpoint.Logger)
	}
	if testEndpoint.ZeroTime != wantZeroTime {
		t.Fatalf("expected: %v, got: %v", wantZeroTime, testEndpoint.ZeroTime)
	}

}

func TestEndpointOptions(t *testing.T) {
	testEndpoint := &Endpoint{}

	for _, opt := range options {
		opt(testEndpoint)
	}

	if testEndpoint.Domain != wantDomain {
		t.Fatalf("expected: %s, got: %s", wantDomain, testEndpoint.Domain)
	}
	if testEndpoint.IDGenerator != wantIDGenerator {
		t.Fatalf("expected: %v, got: %v", wantIDGenerator, testEndpoint.IDGenerator)
	}
	if testEndpoint.Logger != wantLogger {
		t.Fatalf("expected: %v, got: %v", wantLogger, testEndpoint.Logger)
	}
	if testEndpoint.ZeroTime != wantZeroTime {
		t.Fatalf("expected: %v, got: %v", wantZeroTime, testEndpoint.ZeroTime)
	}
}
