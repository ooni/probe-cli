package errorsx

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestErrorWrapperResolverSuccess(t *testing.T) {
	orig := []string{"8.8.8.8"}
	r := &ErrorWrapperResolver{
		Resolver: &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return orig, nil
			},
		},
	}
	addrs, err := r.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != len(orig) || addrs[0] != orig[0] {
		t.Fatal("not the result we expected")
	}
}

func TestErrorWrapperResolverFailure(t *testing.T) {
	r := &ErrorWrapperResolver{
		Resolver: &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return nil, errors.New("no such host")
			},
		},
	}
	ctx := context.Background()
	addrs, err := r.LookupHost(ctx, "dns.google.com")
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
	var errWrapper *ErrWrapper
	if !errors.As(err, &errWrapper) {
		t.Fatal("cannot properly cast the returned error")
	}
	if errWrapper.Failure != FailureDNSNXDOMAINError {
		t.Fatal("unexpected failure")
	}
	if errWrapper.Operation != ResolveOperation {
		t.Fatal("unexpected Operation")
	}
}

func TestErrorWrapperResolverChildNetworkAddress(t *testing.T) {
	r := &ErrorWrapperResolver{Resolver: &mocks.Resolver{
		MockNetwork: func() string {
			return "udp"
		},
		MockAddress: func() string {
			return "8.8.8.8:53"
		},
	}}
	if r.Network() != "udp" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "8.8.8.8:53" {
		t.Fatal("invalid Address")
	}
}

func TestErrorWrapperResolverNoChildNetworkAddress(t *testing.T) {
	r := &ErrorWrapperResolver{Resolver: &net.Resolver{}}
	if r.Network() != "errorWrapper" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}
