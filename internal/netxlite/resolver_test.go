package netxlite

import (
	"context"
	"net"
	"testing"

	"github.com/apex/log"
)

func TestResolverSystemLookupHost(t *testing.T) {
	r := ResolverSystem{}
	if r.Network() != "system" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
	addrs, err := r.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("expected non-nil result here")
	}
}

func TestResolverLoggerWithFailure(t *testing.T) {
	r := ResolverLogger{
		Logger:   log.Log,
		Resolver: DefaultResolver,
	}
	if r.Network() != "system" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
	addrs, err := r.LookupHost(context.Background(), "nonexistent.antani")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
}

func TestResolverLoggerDefaultNetworkAddress(t *testing.T) {
	r := &ResolverLogger{Logger: log.Log, Resolver: &net.Resolver{}}
	if r.Network() != "logger" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}
