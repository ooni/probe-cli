package netxlite

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestResolverLegacyAdapter(t *testing.T) {
	t.Run("with compatible type", func(t *testing.T) {
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
	})

	t.Run("with incompatible type", func(t *testing.T) {
		r := NewResolverLegacyAdapter(&net.Resolver{})
		if r.Network() != "adapter" {
			t.Fatal("invalid Network")
		}
		if r.Address() != "" {
			t.Fatal("invalid Address")
		}
		r.CloseIdleConnections() // does not crash
	})

	t.Run("for LookupHTTPS", func(t *testing.T) {
		r := NewResolverLegacyAdapter(&net.Resolver{})
		https, err := r.LookupHTTPS(context.Background(), "x.org")
		if !errors.Is(err, ErrNoDNSTransport) {
			t.Fatal("not the error we expected")
		}
		if https != nil {
			t.Fatal("expected nil result")
		}
	})
}

func TestDialerLegacyAdapter(t *testing.T) {
	t.Run("with compatible type", func(t *testing.T) {
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
	})

	t.Run("with incompatible type", func(t *testing.T) {
		r := NewDialerLegacyAdapter(&net.Dialer{})
		r.CloseIdleConnections() // does not crash
	})
}

func TestQUICContextDialerAdapter(t *testing.T) {
	t.Run("with compatible type", func(t *testing.T) {
		var called bool
		d := NewQUICDialerFromContextDialerAdapter(&mocks.QUICDialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		})
		d.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("with incompatible type", func(t *testing.T) {
		d := NewQUICDialerFromContextDialerAdapter(&mocks.QUICContextDialer{})
		d.CloseIdleConnections() // does not crash
	})
}
