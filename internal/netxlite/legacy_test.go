package netxlite

import (
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestReduceErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		result := reduceErrors(nil)
		if result != nil {
			t.Fatal("wrong result")
		}
	})
	t.Run("single error", func(t *testing.T) {
		err := errors.New("mocked error")
		result := reduceErrors([]error{err})
		if result != err {
			t.Fatal("wrong result")
		}
	})
	t.Run("multiple errors", func(t *testing.T) {
		err1 := errors.New("mocked error #1")
		err2 := errors.New("mocked error #2")
		result := reduceErrors([]error{err1, err2})
		if result.Error() != "mocked error #1" {
			t.Fatal("wrong result")
		}
	})
	t.Run("multiple errors with meaningful ones", func(t *testing.T) {
		err1 := errors.New("mocked error #1")
		err2 := &errorsx.ErrWrapper{
			Failure: "unknown_failure: antani",
		}
		err3 := &errorsx.ErrWrapper{
			Failure: errorsx.FailureConnectionRefused,
		}
		err4 := errors.New("mocked error #3")
		result := reduceErrors([]error{err1, err2, err3, err4})
		if result.Error() != errorsx.FailureConnectionRefused {
			t.Fatal("wrong result")
		}
	})
}

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
