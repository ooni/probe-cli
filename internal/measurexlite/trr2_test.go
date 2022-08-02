package measurexlite

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
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

	t.Run("with DoH resolver", func(t *testing.T) {
		newResolver := func(model.Logger, string) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"1.1.1.1"}, nil
				},
				MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
					return &model.HTTPSSvc{
						IPv4: []string{"1.1.1.1"},
					}, nil
				},
				MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
					return []*net.NS{
						{
							Host: "1.1.1.1",
						},
					}, nil
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

		t.Run("LookupHTTPS", func(t *testing.T) {
			ctx := context.Background()
			want := &model.HTTPSSvc{
				IPv4: []string{"1.1.1.1"},
			}
			got, err := resolver.LookupHTTPS(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("LookupNS", func(t *testing.T) {
			ctx := context.Background()
			want := &net.NS{
				Host: "1.1.1.1",
			}
			got, err := resolver.LookupNS(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 1 {
				t.Fatal("got unexpected addresses")
			}
			if diff := cmp.Diff(got[0], want); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("DoH resolver times out", func(t *testing.T) {
		var called bool
		newResolver := func(model.DebugLogger, string) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"1.1.1.1"}, nil
				},
				MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
					return &model.HTTPSSvc{
						IPv4: []string{"1.1.1.1"},
					}, nil
				},
				MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
					return []*net.NS{
						{
							Host: "1.1.1.1",
						},
					}, nil
				},
				MockCloseIdleConnections: func() {
					called = true
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

		t.Run("LookupHTTPS", func(t *testing.T) {
			ctx := context.Background()
			want := &model.HTTPSSvc{
				IPv4: []string{"1.1.1.1"},
			}
			got, err := resolver.LookupHTTPS(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("LookupNS", func(t *testing.T) {
			ctx := context.Background()
			want := &net.NS{
				Host: "1.1.1.1",
			}
			got, err := resolver.LookupNS(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 1 {
				t.Fatal("got unexpected addresses")
			}
			if diff := cmp.Diff(got[0], want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("CloseIdleConnections", func(t *testing.T) {
			resolver.CloseIdleConnections()
			if called != true {
				t.Fatal("unexpected error while closing connection")
			}
		})
	})

	t.Run("with system resolver", func(t *testing.T) {
		mockedErr := errors.New("mocked")
		var called bool
		newResolver := func(model.DebugLogger, string) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"1.1.1.1"}, nil
				},
				MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
					return &model.HTTPSSvc{
						IPv4: []string{"1.1.1.1"},
					}, nil
				},
				MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
					return []*net.NS{
						{
							Host: "1.1.1.1",
						},
					}, nil
				},
				MockCloseIdleConnections: func() {
					called = true
				},
			}
		}
		// the DoH resolver must return an error to use the fallback system resolver
		newDoHResolver := func(model.Logger, string) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{}, mockedErr
				},
				MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
					return nil, mockedErr
				},
				MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
					return nil, mockedErr
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

		t.Run("LookupHTTPS", func(t *testing.T) {
			ctx := context.Background()
			want := &model.HTTPSSvc{
				IPv4: []string{"1.1.1.1"},
			}
			got, err := resolver.LookupHTTPS(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("LookupNS", func(t *testing.T) {
			ctx := context.Background()
			want := &net.NS{
				Host: "1.1.1.1",
			}
			got, err := resolver.LookupNS(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 1 {
				t.Fatal("got unexpected addresses")
			}
			if diff := cmp.Diff(got[0], want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("CloseIdleConnections", func(t *testing.T) {
			resolver.CloseIdleConnections()
			if called != true {
				t.Fatal("unexpected error while closing connection")
			}
		})
	})
}
