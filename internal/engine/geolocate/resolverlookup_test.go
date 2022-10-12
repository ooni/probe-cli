package geolocate

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestLookupResolverIP(t *testing.T) {
	rlc := resolverLookupClient{
		Resolver: netxlite.NewStdlibResolver(model.DiscardLogger),
	}
	addr, err := rlc.LookupResolverIP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if addr == "" {
		t.Fatal("expected a non-empty string")
	}
}

type brokenHostLookupper struct {
	err error
}

func (bhl brokenHostLookupper) LookupHost(ctx context.Context, host string) ([]string, error) {
	return nil, bhl.err
}

func TestLookupResolverIPFailure(t *testing.T) {
	expected := errors.New("mocked error")
	rlc := resolverLookupClient{
		Resolver: netxlite.NewStdlibResolver(model.DiscardLogger),
	}
	addr, err := rlc.do(context.Background(), brokenHostLookupper{
		err: expected,
	})
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(addr) != 0 {
		t.Fatal("expected an empty address")
	}
}

func TestLookupResolverIPNoAddressReturned(t *testing.T) {
	rlc := resolverLookupClient{
		Resolver: netxlite.NewStdlibResolver(model.DiscardLogger),
	}
	addr, err := rlc.do(context.Background(), brokenHostLookupper{})
	if !errors.Is(err, ErrNoIPAddressReturned) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(addr) != 0 {
		t.Fatal("expected an empty address")
	}
}
