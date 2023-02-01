package bytecounter

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestMaybeWrapSystemResolver(t *testing.T) {
	t.Run("we don't wrap when the counter is nil", func(t *testing.T) {
		reso := &mocks.Resolver{}
		out := MaybeWrapSystemResolver(reso, nil)
		if reso != out {
			t.Fatal("unexpected out")
		}
	})

	t.Run("Address works as intended", func(t *testing.T) {
		underlying := &mocks.Resolver{
			MockAddress: func() string {
				return "8.8.8.8:53"
			},
		}
		counter := New()
		reso := MaybeWrapSystemResolver(underlying, counter)
		if reso.Address() != "8.8.8.8:53" {
			t.Fatal("unexpected result")
		}
	})

	t.Run("CloseIdleConnections works as intended", func(t *testing.T) {
		var called bool
		underlying := &mocks.Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		counter := New()
		reso := MaybeWrapSystemResolver(underlying, counter)
		reso.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("LookupHTTPS works as intended", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expected := &model.HTTPSSvc{}
			underlying := &mocks.Resolver{
				MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
					return expected, nil
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupHTTPS(context.Background(), "dns.google")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if got != expected {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 10 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 256 {
				t.Fatal("unexpected nrecv")
			}
		})

		t.Run("on non-DNS failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			underlying := &mocks.Resolver{
				MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
					return nil, expected
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupHTTPS(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected error", err)
			}
			if got != nil {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 10 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 0 {
				t.Fatal("unexpected nrecv")
			}
		})

		t.Run("on DNS failure", func(t *testing.T) {
			expected := errors.New(netxlite.FailureDNSNXDOMAINError)
			underlying := &mocks.Resolver{
				MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
					return nil, expected
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupHTTPS(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected error", err)
			}
			if got != nil {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 10 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 128 {
				t.Fatal("unexpected nrecv")
			}
		})
	})

	t.Run("LookupNS works as intended", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			underlying := &mocks.Resolver{
				MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
					out := make([]*net.NS, 3)
					return out, nil
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupNS(context.Background(), "dns.google")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 3 {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 10 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 256 {
				t.Fatal("unexpected nrecv")
			}
		})

		t.Run("on non-DNS failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			underlying := &mocks.Resolver{
				MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
					return nil, expected
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupNS(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 0 {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 10 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 0 {
				t.Fatal("unexpected nrecv")
			}
		})

		t.Run("on DNS failure", func(t *testing.T) {
			expected := errors.New(netxlite.FailureDNSNXDOMAINError)
			underlying := &mocks.Resolver{
				MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
					return nil, expected
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupNS(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 0 {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 10 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 128 {
				t.Fatal("unexpected nrecv")
			}
		})
	})

	t.Run("LookupHost works as intended", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			underlying := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					out := make([]string, 3)
					return out, nil
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupHost(context.Background(), "dns.google")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 3 {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 20 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 256 {
				t.Fatal("unexpected nrecv")
			}
		})

		t.Run("on non-DNS failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			underlying := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, expected
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupHost(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 0 {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 20 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 0 {
				t.Fatal("unexpected nrecv")
			}
		})

		t.Run("on DNS failure", func(t *testing.T) {
			expected := errors.New(netxlite.FailureDNSNXDOMAINError)
			underlying := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, expected
				},
			}
			counter := New()
			reso := MaybeWrapSystemResolver(underlying, counter)
			got, err := reso.LookupHost(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 0 {
				t.Fatal("invalid result")
			}
			if nsent := counter.BytesSent(); nsent != 20 {
				t.Fatal("unexpected nsent", nsent)
			}
			if nrecv := counter.BytesReceived(); nrecv != 128 {
				t.Fatal("unexpected nrecv")
			}
		})
	})

	t.Run("Network works as intended", func(t *testing.T) {
		underlying := &mocks.Resolver{
			MockNetwork: func() string {
				return "udp"
			},
		}
		counter := New()
		reso := MaybeWrapSystemResolver(underlying, counter)
		if reso.Network() != "udp" {
			t.Fatal("unexpected result")
		}
	})
}
