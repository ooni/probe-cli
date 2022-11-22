package geolocate

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestLookupResolverIP(t *testing.T) {
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
	expected := errors.New("mocked error")
	rlc := resolverLookupClient{
		Logger: model.DiscardLogger,
	}

	// Note well: because we want to really enforce the implementation of the
	// resolverlookup to use the system resolver, here we are using TProxy for
	// testing rather than having a mockable resolver as we normally do.
	//
	// We're doing this because we want to make it less likely that we will
	// introduce bug https://github.com/ooni/probe/issues/2360 again.
	oldTProxy := netxlite.TProxy
	defer func() {
		netxlite.TProxy = oldTProxy
	}()
	netxlite.TProxy = &mocks.UnderlyingNetwork{
		MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
			return nil, "", expected
		},
		MockGetaddrinfoResolverNetwork: func() string {
			return netxlite.StdlibResolverGetaddrinfo
		},
	}

	addr, err := rlc.LookupResolverIP(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(addr) != 0 {
		t.Fatal("expected an empty address")
	}
}

func TestLookupResolverIPNoAddressReturned(t *testing.T) {
	rlc := resolverLookupClient{
		Logger: model.DiscardLogger,
	}

	// Note well: because we want to really enforce the implementation of the
	// resolverlookup to use the system resolver, here we are using TProxy for
	// testing rather than having a mockable resolver as we normally do.
	//
	// We're doing this because we want to make it less likely that we will
	// introduce bug https://github.com/ooni/probe/issues/2360 again.
	oldTProxy := netxlite.TProxy
	defer func() {
		netxlite.TProxy = oldTProxy
	}()
	netxlite.TProxy = &mocks.UnderlyingNetwork{
		MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
			return nil, "", nil
		},
		MockGetaddrinfoResolverNetwork: func() string {
			return netxlite.StdlibResolverGetaddrinfo
		},
	}

	addr, err := rlc.LookupResolverIP(context.Background())
	if err == nil || err.Error() != netxlite.FailureDNSNoAnswer {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(addr) != 0 {
		t.Fatal("expected an empty address")
	}
}
