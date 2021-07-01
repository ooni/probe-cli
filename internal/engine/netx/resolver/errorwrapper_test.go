package resolver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

func TestErrorWrapperSuccess(t *testing.T) {
	orig := []string{"8.8.8.8"}
	r := resolver.ErrorWrapperResolver{
		Resolver: resolver.NewFakeResolverWithResult(orig),
	}
	addrs, err := r.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != len(orig) || addrs[0] != orig[0] {
		t.Fatal("not the result we expected")
	}
}

func TestErrorWrapperFailure(t *testing.T) {
	r := resolver.ErrorWrapperResolver{
		Resolver: resolver.NewFakeResolverThatFails(),
	}
	ctx := context.Background()
	addrs, err := r.LookupHost(ctx, "dns.google.com")
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
	var errWrapper *errorsx.ErrWrapper
	if !errors.As(err, &errWrapper) {
		t.Fatal("cannot properly cast the returned error")
	}
	if errWrapper.Failure != errorsx.FailureDNSNXDOMAINError {
		t.Fatal("unexpected failure")
	}
	if errWrapper.Operation != errorsx.ResolveOperation {
		t.Fatal("unexpected Operation")
	}
}
