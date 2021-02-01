package resolver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
)

func TestCacheFailure(t *testing.T) {
	expected := errors.New("mocked error")
	var r resolver.Resolver = resolver.FakeResolver{
		Err: expected,
	}
	cache := &resolver.CacheResolver{Resolver: r}
	addrs, err := cache.LookupHost(context.Background(), "www.google.com")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs here")
	}
	if cache.Get("www.google.com") != nil {
		t.Fatal("expected empty cache here")
	}
}

func TestCacheHitSuccess(t *testing.T) {
	var r resolver.Resolver = resolver.FakeResolver{
		Err: errors.New("mocked error"),
	}
	cache := &resolver.CacheResolver{Resolver: r}
	cache.Set("dns.google.com", []string{"8.8.8.8"})
	addrs, err := cache.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
		t.Fatal("not the result we expected")
	}
}

func TestCacheMissSuccess(t *testing.T) {
	var r resolver.Resolver = resolver.FakeResolver{
		Result: []string{"8.8.8.8"},
	}
	cache := &resolver.CacheResolver{Resolver: r}
	addrs, err := cache.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
		t.Fatal("not the result we expected")
	}
	if cache.Get("dns.google.com")[0] != "8.8.8.8" {
		t.Fatal("expected full cache here")
	}
}

func TestCacheReadonlySuccess(t *testing.T) {
	var r resolver.Resolver = resolver.FakeResolver{
		Result: []string{"8.8.8.8"},
	}
	cache := &resolver.CacheResolver{Resolver: r, ReadOnly: true}
	addrs, err := cache.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
		t.Fatal("not the result we expected")
	}
	if cache.Get("dns.google.com") != nil {
		t.Fatal("expected empty cache here")
	}
}
