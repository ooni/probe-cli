package netxlite

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestResolverSystemNetworkAddress(t *testing.T) {
	r := resolverSystem{}
	if r.Network() != "system" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}

func TestResolverSystemWorksAsIntended(t *testing.T) {
	r := resolverSystem{}
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
	r := resolverLogger{
		Logger: log.Log,
		Resolver: &mocks.Resolver{
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
	r := resolverLogger{
		Logger: log.Log,
		Resolver: &mocks.Resolver{
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
	r := &resolverLogger{Logger: log.Log, Resolver: DefaultResolver}
	if r.Network() != "system" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}

func TestResolverLoggerNoChildNetworkAddress(t *testing.T) {
	r := &resolverLogger{Logger: log.Log, Resolver: &net.Resolver{}}
	if r.Network() != "logger" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}

func TestResolverIDNAWorksAsIntended(t *testing.T) {
	expectedIPs := []string{"77.88.55.66"}
	r := &resolverIDNA{
		Resolver: &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				if domain != "xn--d1acpjx3f.xn--p1ai" {
					return nil, errors.New("passed invalid domain")
				}
				return expectedIPs, nil
			},
		},
	}
	ctx := context.Background()
	addrs, err := r.LookupHost(ctx, "яндекс.рф")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedIPs, addrs); diff != "" {
		t.Fatal(diff)
	}
}

func TestResolverIDNAWithInvalidPunycode(t *testing.T) {
	r := &resolverIDNA{Resolver: &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, errors.New("should not happen")
		},
	}}
	// See https://www.farsightsecurity.com/blog/txt-record/punycode-20180711/
	ctx := context.Background()
	addrs, err := r.LookupHost(ctx, "xn--0000h")
	if err == nil || !strings.HasPrefix(err.Error(), "idna: invalid label") {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected no response here")
	}
}

func TestResolverIDNAChildNetworkAddress(t *testing.T) {
	r := &resolverIDNA{
		Resolver: DefaultResolver,
	}
	if v := r.Network(); v != "system" {
		t.Fatal("invalid network", v)
	}
	if v := r.Address(); v != "" {
		t.Fatal("invalid address", v)
	}
}

func TestResolverIDNANoChildNetworkAddress(t *testing.T) {
	r := &resolverIDNA{}
	if v := r.Network(); v != "idna" {
		t.Fatal("invalid network", v)
	}
	if v := r.Address(); v != "" {
		t.Fatal("invalid address", v)
	}
}
