package mocks

import (
	"context"
	"errors"
	"testing"
)

func TestResolverLookupHost(t *testing.T) {
	expected := errors.New("mocked error")
	r := &Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, expected
		},
	}
	ctx := context.Background()
	addrs, err := r.LookupHost(ctx, "dns.google")
	if !errors.Is(err, expected) {
		t.Fatal("unexpected error", err)
	}
	if addrs != nil {
		t.Fatal("expected nil addr")
	}
}

func TestResolverNetwork(t *testing.T) {
	r := &Resolver{
		MockNetwork: func() string {
			return "antani"
		},
	}
	if v := r.Network(); v != "antani" {
		t.Fatal("unexpected network", v)
	}
}

func TestResolverAddress(t *testing.T) {
	r := &Resolver{
		MockAddress: func() string {
			return "1.1.1.1"
		},
	}
	if v := r.Address(); v != "1.1.1.1" {
		t.Fatal("unexpected address", v)
	}
}
