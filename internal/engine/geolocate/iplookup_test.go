package geolocate

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestIPLookupGood(t *testing.T) {
	ip, err := (ipLookupClient{
		Logger:    log.Log,
		Resolver:  netxlite.NewResolverStdlib(model.DiscardLogger),
		UserAgent: "ooniprobe-engine/0.1.0",
	}).LookupProbeIP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatal("not an IP address")
	}
}

func TestIPLookupAllFailed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel to cause Do() to fail
	ip, err := (ipLookupClient{
		Logger:    log.Log,
		Resolver:  netxlite.NewResolverStdlib(model.DiscardLogger),
		UserAgent: "ooniprobe-engine/0.1.0",
	}).LookupProbeIP(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("expected an error here")
	}
	if ip != DefaultProbeIP {
		t.Fatal("expected the default IP here")
	}
}

func TestIPLookupInvalidIP(t *testing.T) {
	ctx := context.Background()
	ip, err := (ipLookupClient{
		Logger:    log.Log,
		Resolver:  netxlite.NewResolverStdlib(model.DiscardLogger),
		UserAgent: "ooniprobe-engine/0.1.0",
	}).doWithCustomFunc(ctx, invalidIPLookup)
	if !errors.Is(err, ErrInvalidIPAddress) {
		t.Fatal("expected an error here")
	}
	if ip != DefaultProbeIP {
		t.Fatal("expected the default IP here")
	}
}
