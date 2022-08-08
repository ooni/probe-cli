package measurexlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewTrustedRecursiveResolver(t *testing.T) {
	t.Run("default URL", func(t *testing.T) {
		expected := "https://mozilla.cloudflare-dns.com/dns-query"
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "", 0).(*TrustedRecursiveResolver2)
		if resolver.URL != expected {
			t.Fatal("unexpected default url")
		}
	})

	t.Run("custom URL", func(t *testing.T) {
		url := "https://dns.google/dns-query"
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, url, 0).(*TrustedRecursiveResolver2)
		if resolver.URL != "https://dns.google/dns-query" {
			t.Fatal("unexpected default url")
		}
	})

	t.Run("default timeout", func(t *testing.T) {
		expected := 1500 * time.Millisecond
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "", 0).(*TrustedRecursiveResolver2)
		if resolver.Timeout != expected {
			t.Fatal("unexpected default timeout")
		}
	})

	t.Run("custom timeout", func(t *testing.T) {
		expected := 10 * time.Millisecond
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "", 10).(*TrustedRecursiveResolver2)
		if resolver.Timeout != expected {
			t.Fatal("unexpected default timeout")
		}
	})

	t.Run("network gives correct output", func(t *testing.T) {
		expected := "trr2"
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "", 0).(*TrustedRecursiveResolver2)
		if resolver.Network() != expected {
			t.Fatal("network gives unexpected output")
		}
	})

	t.Run("NewParallelDNSOverHTTPSResolverFn is nil", func(t *testing.T) {
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "", 0).(*TrustedRecursiveResolver2)
		if resolver.NewParallelDNSOverHTTPSResolverFn != nil {
			t.Fatal("expected nil NewParallelDNSOverHTTPSResolverFn")
		}
	})

	t.Run("NewParallelDNSOverHTTPSResolverFn works as intended", func(t *testing.T) {
		t.Run("when not nil", func(t *testing.T) {
			underlying := &mocks.Resolver{}
			resolver := &TrustedRecursiveResolver2{
				NewParallelDNSOverHTTPSResolverFn: func() model.Resolver {
					return underlying
				},
			}
			got := resolver.newParallelDNSOverHTTPSResolver(log.Log, "")
			if got != underlying {
				t.Fatal("unexpected parallel DoH resolver")
			}

		})

		t.Run("when nil", func(t *testing.T) {
			resolver := &TrustedRecursiveResolver2{}
			got := resolver.newParallelDNSOverHTTPSResolver(model.DiscardLogger, "dns.google.com")
			if got.Network() != "doh" {
				t.Fatal("unexpected resolver network")
			}
		})
	})

	t.Run("with DoH resolver", func(t *testing.T) {
		newResolver := func() model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"1.1.1.1"}, nil
				},
			}
		}
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "", 0).(*TrustedRecursiveResolver2)
		resolver.NewParallelDNSOverHTTPSResolverFn = newResolver

		t.Run("LookupHost", func(t *testing.T) {
			ctx := context.Background()
			addrs, err := resolver.LookupHost(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
				t.Fatal("got unexpected addresses")
			}
		})
	})

	t.Run("DoH resolver times out", func(t *testing.T) {
		newResolver := func(model.DebugLogger, string) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"1.1.1.1"}, nil
				},
			}
		}
		// force the context to time out
		resolver := NewTrustedRecursiveResolver2(log.Log, "", 10).(*TrustedRecursiveResolver2)
		resolver.ResolverSystem = newResolver(log.Log, "")

		t.Run("LookupHost", func(t *testing.T) {
			ctx := context.Background()
			addrs, err := resolver.LookupHost(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
				t.Fatal("got unexpected addresses")
			}
		})
	})

	t.Run("with system resolver", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		newResolver := func(model.DebugLogger, string) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"1.1.1.1"}, nil
				},
			}
		}
		// the DoH resolver must return an error to use the fallback system resolver
		newDoHResolver := func() model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{}, mockedErr
				},
			}
		}
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "", 0).(*TrustedRecursiveResolver2)
		resolver.ResolverSystem = newResolver(log.Log, "")
		resolver.NewParallelDNSOverHTTPSResolverFn = newDoHResolver

		t.Run("LookupHost", func(t *testing.T) {
			ctx := context.Background()
			addrs, err := resolver.LookupHost(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
				t.Fatal("got unexpected addresses")
			}
		})
	})
}
