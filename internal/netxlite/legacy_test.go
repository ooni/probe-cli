package netxlite

import (
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestResolverLegacyAdapterWithCompatibleType(t *testing.T) {
	var called bool
	r := NewResolverLegacyAdapter(&mocks.Resolver{
		MockNetwork: func() string {
			return "network"
		},
		MockAddress: func() string {
			return "address"
		},
		MockCloseIdleConnections: func() {
			called = true
		},
	})
	if r.Network() != "network" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "address" {
		t.Fatal("invalid Address")
	}
	r.CloseIdleConnections()
	if !called {
		t.Fatal("not called")
	}
}

func TestResolverLegacyAdapterDefaults(t *testing.T) {
	r := NewResolverLegacyAdapter(&net.Resolver{})
	if r.Network() != "adapter" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
	r.CloseIdleConnections() // does not crash
}

func TestDialerLegacyAdapterWithCompatibleType(t *testing.T) {
	var called bool
	r := NewDialerLegacyAdapter(&mocks.Dialer{
		MockCloseIdleConnections: func() {
			called = true
		},
	})
	r.CloseIdleConnections()
	if !called {
		t.Fatal("not called")
	}
}

func TestDialerLegacyAdapterDefaults(t *testing.T) {
	r := NewDialerLegacyAdapter(&net.Dialer{})
	r.CloseIdleConnections() // does not crash
}
