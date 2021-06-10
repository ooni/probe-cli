package resolver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/dialid"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/transactionid"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
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
	ctx = dialid.WithDialID(ctx)
	ctx = transactionid.WithTransactionID(ctx)
	addrs, err := r.LookupHost(ctx, "dns.google.com")
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
	var resolveErr resolver.ErrResolve
	if !errors.As(err, &resolveErr) {
		t.Fatal("cannot properly cast the returned error")
	}
	if *archival.NewFailure(err) != errorx.FailureDNSNXDOMAINError {
		t.Fatal("unexpected failure")
	}
	if *archival.NewFailedOperation(err) != errorx.ResolveOperation {
		t.Fatal("unexpected Operation", *archival.NewFailedOperation(err))
	}
}
