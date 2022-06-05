package netxlite

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestMaybeWrapWithCachingResolver(t *testing.T) {
	t.Run("with enable equal to true", func(t *testing.T) {
		underlying := &mocks.Resolver{}
		reso := MaybeWrapWithCachingResolver(true, underlying)
		cachereso := reso.(*cacheResolver)
		if cachereso.resolver != underlying {
			t.Fatal("did not wrap correctly")
		}
	})

	t.Run("with enable equal to false", func(t *testing.T) {
		underlying := &mocks.Resolver{}
		reso := MaybeWrapWithCachingResolver(false, underlying)
		if reso != underlying {
			t.Fatal("unexpected result")
		}
	})
}

func TestMaybeWrapWithStaticDNSCache(t *testing.T) {
	t.Run("when the cache is not empty", func(t *testing.T) {
		cachedDomain := "dns.google"
		expectedEntry := []string{"8.8.8.8", "8.8.4.4"}
		underlyingCache := make(map[string][]string)
		underlyingCache[cachedDomain] = expectedEntry
		underlyingReso := &mocks.Resolver{}
		reso := MaybeWrapWithStaticDNSCache(underlyingCache, underlyingReso)
		cachereso := reso.(*cacheResolver)
		if diff := cmp.Diff(cachereso.cache, underlyingCache); diff != "" {
			t.Fatal(diff)
		}
		if cachereso.resolver != underlyingReso {
			t.Fatal("unexpected underlying resolver")
		}
	})

	t.Run("when the cache is empty", func(t *testing.T) {
		underlyingCache := make(map[string][]string)
		underlyingReso := &mocks.Resolver{}
		reso := MaybeWrapWithStaticDNSCache(underlyingCache, underlyingReso)
		if reso != underlyingReso {
			t.Fatal("unexpected result")
		}
	})

	t.Run("when the cache is nil", func(t *testing.T) {
		var underlyingCache map[string][]string
		underlyingReso := &mocks.Resolver{}
		reso := MaybeWrapWithStaticDNSCache(underlyingCache, underlyingReso)
		if reso != underlyingReso {
			t.Fatal("unexpected result")
		}
	})
}

func TestCacheResolver(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("cache miss and failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, expected
				},
			}
			cache := &cacheResolver{resolver: r}
			addrs, err := cache.LookupHost(context.Background(), "www.google.com")
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
			if addrs != nil {
				t.Fatal("expected nil addrs here")
			}
			if cache.get("www.google.com") != nil {
				t.Fatal("expected empty cache here")
			}
		})

		t.Run("cache hit", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, expected
				},
			}
			cache := &cacheResolver{resolver: r}
			cache.set("dns.google.com", []string{"8.8.8.8"})
			addrs, err := cache.LookupHost(context.Background(), "dns.google.com")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
				t.Fatal("not the result we expected")
			}
		})

		t.Run("cache miss and success with readwrite cache", func(t *testing.T) {
			r := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"8.8.8.8"}, nil
				},
			}
			cache := &cacheResolver{resolver: r}
			addrs, err := cache.LookupHost(context.Background(), "dns.google.com")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
				t.Fatal("not the result we expected")
			}
			if cache.get("dns.google.com")[0] != "8.8.8.8" {
				t.Fatal("expected full cache here")
			}
		})

		t.Run("cache miss and success with readonly cache", func(t *testing.T) {
			r := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"8.8.8.8"}, nil
				},
			}
			cache := &cacheResolver{resolver: r, readOnly: true}
			addrs, err := cache.LookupHost(context.Background(), "dns.google.com")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
				t.Fatal("not the result we expected")
			}
			if cache.get("dns.google.com") != nil {
				t.Fatal("expected empty cache here")
			}
		})

		t.Run("Address", func(t *testing.T) {
			underlying := &mocks.Resolver{
				MockAddress: func() string {
					return "x"
				},
			}
			reso := &cacheResolver{resolver: underlying}
			if reso.Address() != "x" {
				t.Fatal("unexpected result")
			}
		})

		t.Run("Network", func(t *testing.T) {
			underlying := &mocks.Resolver{
				MockNetwork: func() string {
					return "x"
				},
			}
			reso := &cacheResolver{resolver: underlying}
			if reso.Network() != "x" {
				t.Fatal("unexpected result")
			}
		})

		t.Run("CloseIdleConnections", func(t *testing.T) {
			var called bool
			underlying := &mocks.Resolver{
				MockCloseIdleConnections: func() {
					called = true
				},
			}
			reso := &cacheResolver{resolver: underlying}
			reso.CloseIdleConnections()
			if !called {
				t.Fatal("not called")
			}
		})

		t.Run("LookupHTTPS", func(t *testing.T) {
			reso := &cacheResolver{}
			https, err := reso.LookupHTTPS(context.Background(), "dns.google")
			if !errors.Is(err, ErrNoDNSTransport) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("expected nil")
			}
		})

		t.Run("LookupNS", func(t *testing.T) {
			reso := &cacheResolver{}
			ns, err := reso.LookupNS(context.Background(), "dns.google")
			if !errors.Is(err, ErrNoDNSTransport) {
				t.Fatal("unexpected err", err)
			}
			if len(ns) != 0 {
				t.Fatal("expected zero length slice")
			}
		})
	})
}
