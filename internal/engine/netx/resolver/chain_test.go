package resolver_test

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestChainLookupHost(t *testing.T) {
	r := resolver.ChainResolver{
		Primary:   resolver.NewFakeResolverThatFails(),
		Secondary: &netxlite.ResolverSystem{},
	}
	if r.Address() != "" {
		t.Fatal("invalid address")
	}
	if r.Network() != "chain" {
		t.Fatal("invalid network")
	}
	addrs, err := r.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("expect non nil return value here")
	}
}
