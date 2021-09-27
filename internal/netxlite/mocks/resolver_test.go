package mocks

import (
	"context"
	"errors"
	"testing"
)

func TestResolver(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
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
	})

	t.Run("Network", func(t *testing.T) {
		r := &Resolver{
			MockNetwork: func() string {
				return "antani"
			},
		}
		if v := r.Network(); v != "antani" {
			t.Fatal("unexpected network", v)
		}
	})

	t.Run("Address", func(t *testing.T) {
		r := &Resolver{
			MockAddress: func() string {
				return "1.1.1.1"
			},
		}
		if v := r.Address(); v != "1.1.1.1" {
			t.Fatal("unexpected address", v)
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		r := &Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		r.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		expected := errors.New("mocked error")
		r := &Resolver{
			MockLookupHTTPS: func(ctx context.Context, domain string) (*HTTPSSvc, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		https, err := r.LookupHTTPS(ctx, "dns.google")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if https != nil {
			t.Fatal("expected nil addr")
		}
	})
}
