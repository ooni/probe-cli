package resolver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
)

func TestAddressSuccess(t *testing.T) {
	r := resolver.AddressResolver{}
	addrs, err := r.LookupHost(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
		t.Fatal("not the result we expected")
	}
}

func TestAddressFailure(t *testing.T) {
	expected := errors.New("mocked error")
	r := resolver.AddressResolver{
		Resolver: resolver.FakeResolver{
			Err: expected,
		},
	}
	addrs, err := r.LookupHost(context.Background(), "dns.google.com")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs")
	}
}
