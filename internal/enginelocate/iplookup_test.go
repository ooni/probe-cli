package enginelocate

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestIPLookupGood(t *testing.T) {
	ip, err := (ipLookupClient{
		Logger:    log.Log,
		Resolver:  netxlite.NewStdlibResolver(model.DiscardLogger),
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
		Resolver:  netxlite.NewStdlibResolver(model.DiscardLogger),
		UserAgent: "ooniprobe-engine/0.1.0",
	}).LookupProbeIP(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("expected an error here")
	}
	if ip != model.DefaultProbeIP {
		t.Fatal("expected the default IP here")
	}
}

func TestIPLookupInvalidIP(t *testing.T) {
	ctx := context.Background()
	ip, err := (ipLookupClient{
		Logger:    log.Log,
		Resolver:  netxlite.NewStdlibResolver(model.DiscardLogger),
		UserAgent: "ooniprobe-engine/0.1.0",
	}).doWithCustomFunc(ctx, invalidIPLookup)
	if !errors.Is(err, ErrInvalidIPAddress) {
		t.Fatal("expected an error here")
	}
	if ip != model.DefaultProbeIP {
		t.Fatal("expected the default IP here")
	}
}

func TestContextForIPLookupWithTimeout(t *testing.T) {
	now := time.Now()
	ctx, cancel := contextForIPLookupWithTimeout(context.Background())
	defer cancel()
	deadline, okay := ctx.Deadline()
	if !okay {
		t.Fatal("the context does not have a deadline")
	}
	delta := deadline.Sub(now)
	// Note: super conservative check. Assume it may take up to five seconds
	// for the code to create a context, which is totally unrealistic.
	if delta < 40*time.Second {
		t.Fatal("the deadline is too short")
	}
}
