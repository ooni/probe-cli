package measurexlite

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewTrustedRecursiveResolver(t *testing.T) {
	var called bool
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
			MockCloseIdleConnections: func() {
				called = true
			},
		}
	}

	t.Run("default URL", func(t *testing.T) {
		expected := "https://mozilla.cloudflare-dns.com/dns-query"
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "")
		resolvert := resolver.(*TrustedRecursiveResolver2)
		if resolvert.Url() != expected {
			t.Fatal("unexpected default url")
		}
	})

	t.Run("custom URL", func(t *testing.T) {
		expected := "https://dns.google/dns-query"
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "https://dns.google/dns-query")
		resolvert := resolver.(*TrustedRecursiveResolver2)
		if resolvert.Url() != expected {
			t.Fatal("unexpected default url")
		}
	})

	t.Run("with DoH resolver", func(t *testing.T) {
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "")
		resolvert := resolver.(*TrustedRecursiveResolver2)
		resolvert.NewParallelDNSOverHTTPSResolverFn = newResolver
		ctx := context.Background()
		t.Run("LookupHost", func(t *testing.T) {
			addrs, err := resolver.LookupHost(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
				t.Fatal("got unexpected addresses")
			}
		})
		t.Run("LookupHTTPS", func(t *testing.T) {
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
	t.Run("with system resolver", func(t *testing.T) {
		mockedErr := errors.New("mocked")
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
		resolver := NewTrustedRecursiveResolver2(&mocks.Logger{}, "")
		resolvert := resolver.(*TrustedRecursiveResolver2)
		resolvert.NewParallelDNSOverHTTPSResolverFn = newDoHResolver
		resolvert.ResolverSystem = newResolver(&mocks.Logger{}, "")
		ctx := context.Background()
		t.Run("LookupHost", func(t *testing.T) {
			addrs, err := resolver.LookupHost(ctx, "example.com")
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
				t.Fatal("got unexpected addresses")
			}
		})
		t.Run("LookupHTTPS", func(t *testing.T) {
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
				t.Fatal("unexpected error while calling CloseIdleConnections")
			}
		})
	})
}
