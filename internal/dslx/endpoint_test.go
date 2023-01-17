package dslx

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/apex/log"
)

var wantEndpoint *Endpoint = &Endpoint{
	Domain:      "www.google.com",
	IDGenerator: &atomic.Int64{},
	Logger:      log.Log,
	ZeroTime:    time.Time{},
}

var options []EndpointOption = []EndpointOption{
	EndpointOptionDomain(wantEndpoint.Domain),
	EndpointOptionIDGenerator(wantEndpoint.IDGenerator),
	EndpointOptionLogger(wantEndpoint.Logger),
	EndpointOptionZeroTime(wantEndpoint.ZeroTime),
}

func TestNewEndpoint(t *testing.T) {
	testEndpoint := NewEndpoint("network", "a.b.c.d", options...)

	if testEndpoint.Network != "network" {
		t.Fatalf("expected: %s, got: %s", "network", testEndpoint.Network)
	}
	if testEndpoint.Address != "a.b.c.d" {
		t.Fatalf("expected: %s, got: %s", "a.b.c.d", testEndpoint.Address)
	}
	if testEndpoint.Domain != wantEndpoint.Domain {
		t.Fatalf("expected: %s, got: %s", wantEndpoint.Domain, testEndpoint.Domain)
	}
	if testEndpoint.IDGenerator != wantEndpoint.IDGenerator {
		t.Fatalf("expected: %v, got: %v", wantEndpoint.IDGenerator, testEndpoint.IDGenerator)
	}
	if testEndpoint.Logger != wantEndpoint.Logger {
		t.Fatalf("expected: %v, got: %v", wantEndpoint.Logger, testEndpoint.Logger)
	}
	if testEndpoint.ZeroTime != wantEndpoint.ZeroTime {
		t.Fatalf("expected: %v, got: %v", wantEndpoint.ZeroTime, testEndpoint.ZeroTime)
	}

}

func TestEndpointOptions(t *testing.T) {
	testEndpoint := &Endpoint{}

	for _, opt := range options {
		opt(testEndpoint)
	}

	if testEndpoint.Domain != wantEndpoint.Domain {
		t.Fatalf("expected: %s, got: %s", wantEndpoint.Domain, testEndpoint.Domain)
	}
	if testEndpoint.IDGenerator != wantEndpoint.IDGenerator {
		t.Fatalf("expected: %v, got: %v", wantEndpoint.IDGenerator, testEndpoint.IDGenerator)
	}
	if testEndpoint.Logger != wantEndpoint.Logger {
		t.Fatalf("expected: %v, got: %v", wantEndpoint.Logger, testEndpoint.Logger)
	}
	if testEndpoint.ZeroTime != wantEndpoint.ZeroTime {
		t.Fatalf("expected: %v, got: %v", wantEndpoint.ZeroTime, testEndpoint.ZeroTime)
	}
}
