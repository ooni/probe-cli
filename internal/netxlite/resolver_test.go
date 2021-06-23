package netxlite

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxmocks"
)

func TestResolverSystemNetworkAddress(t *testing.T) {
	r := ResolverSystem{}
	if r.Network() != "system" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}

func TestResolverSystemWorksAsIntended(t *testing.T) {
	r := ResolverSystem{}
	addrs, err := r.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("expected non-nil result here")
	}
}

func TestResolverLoggerWithSuccess(t *testing.T) {
	expected := []string{"1.1.1.1"}
	r := ResolverLogger{
		Logger: log.Log,
		Resolver: &netxmocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return expected, nil
			},
		},
	}
	addrs, err := r.LookupHost(context.Background(), "dns.google")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, addrs); diff != "" {
		t.Fatal(diff)
	}
}

func TestResolverLoggerWithFailure(t *testing.T) {
	expected := errors.New("mocked error")
	r := ResolverLogger{
		Logger: log.Log,
		Resolver: &netxmocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return nil, expected
			},
		},
	}
	addrs, err := r.LookupHost(context.Background(), "dns.google")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
}

func TestResolverLoggerChildNetworkAddress(t *testing.T) {
	r := &ResolverLogger{Logger: log.Log, Resolver: DefaultResolver}
	if r.Network() != "system" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}

func TestResolverLoggerNoChildNetworkAddress(t *testing.T) {
	r := &ResolverLogger{Logger: log.Log, Resolver: &net.Resolver{}}
	if r.Network() != "logger" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}
