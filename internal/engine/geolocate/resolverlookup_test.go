package geolocate

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestLookupResolverIPSuccess(t *testing.T) {
	rlc := resolverLookupClient{
		Logger: model.DiscardLogger,
	}
	addr, err := rlc.LookupResolverIP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if addr == "" {
		t.Fatal("expected a non-empty string")
	}
}

func TestLookupResolverIPFailure(t *testing.T) {
	rlc := resolverLookupClient{
		Logger: model.DiscardLogger,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // stop immediately
	addr, err := rlc.LookupResolverIP(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(addr) != 0 {
		t.Fatal("expected an empty address")
	}
}
