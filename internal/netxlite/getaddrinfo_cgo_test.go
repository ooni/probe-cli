//go:build: cgo

package netxlite

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestGetaddrinfoLookupANYWithIPAddressInsteadOfDomain(t *testing.T) {
	// Getaddrinfo also works when the domain to resolve is actually an IP address
	// so we can ensure it's working without hitting the network.
	addrs, cname, err := getaddrinfoLookupANY(context.Background(), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	foundA, foundAAAA := false, false
	for _, addr := range addrs {
		foundA = foundA || addr == "127.0.0.1"
		foundAAAA = foundAAAA || addr == "::1"
	}
	if !foundA && !foundAAAA {
		t.Fatal("it seems we cannot resolve localhost", addrs)
	}
	if cname != "localhost" && cname != "localhost6" {
		t.Fatal("there seems to be no CNAME for localhost", cname)
	}
}

func TestGetaddrinfoLookupANYWithNoDomain(t *testing.T) {
	addresses, cname, err := getaddrinfoLookupANY(context.Background(), "")
	if !errors.Is(err, ErrOODNSNoSuchHost) {
		t.Fatal("unexpected err", err)
	}
	if len(addresses) > 0 {
		t.Fatal("expected no addresses", addresses)
	}
	if cname != "" {
		t.Fatal("expected empty cname", cname)
	}
}

func TestGetaddrinfoStateAddrinfoToStringWithInvalidFamily(t *testing.T) {
	aip := staticAddrinfoWithInvalidFamily()
	state := newGetaddrinfoState(getaddrinfoNumSlots)
	addr, err := state.addrinfoToString(aip)
	if !errors.Is(err, errGetaddrinfoUnknownFamily) {
		t.Fatal("unexpected err", err)
	}
	if addr != "" {
		t.Fatal("expected empty addr here")
	}
}

func TestGetaddrinfoStateIfnametoindex(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	state := newGetaddrinfoState(getaddrinfoNumSlots)
	for _, iface := range ifaces {
		name := state.ifnametoindex(iface.Index)
		if name != iface.Name {
			t.Fatal("unexpected name")
		}
	}
}

func TestGetaddrinfoStateLookupANYWithNoSlots(t *testing.T) {
	const (
		noslots = 0
		timeout = 10 * time.Millisecond
	)
	state := newGetaddrinfoState(noslots)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	addresses, cname, err := state.LookupANY(ctx, "dns.google")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("unexpected err", err)
	}
	if len(addresses) > 0 {
		t.Fatal("expected no addresses", addresses)
	}
	if cname != "" {
		t.Fatal("expected empty cname", cname)
	}
}

func TestGetaddrinfoStateToAddressList(t *testing.T) {
	t.Run("with invalid sockety type", func(t *testing.T) {
		state := newGetaddrinfoState(0) // number of slots not relevant
		aip := staticAddrinfoWithInvalidSocketType()
		addresses, cname, err := state.toAddressList(aip)
		if !errors.Is(err, ErrOODNSNoAnswer) {
			t.Fatal("unexpected err", err)
		}
		if len(addresses) > 0 {
			t.Fatal("expected no addresses", addresses)
		}
		if cname != "" {
			t.Fatal("expected empty cname", cname)
		}
	})

	t.Run("with invalid family", func(t *testing.T) {
		state := newGetaddrinfoState(0) // number of slots not relevant
		aip := staticAddrinfoWithInvalidFamily()
		addresses, cname, err := state.toAddressList(aip)
		if !errors.Is(err, ErrOODNSNoAnswer) {
			t.Fatal("unexpected err", err)
		}
		if len(addresses) > 0 {
			t.Fatal("expected no addresses", addresses)
		}
		if cname != "" {
			t.Fatal("expected empty cname", cname)
		}
	})
}
