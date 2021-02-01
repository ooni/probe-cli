package geolocate

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
)

func TestIPLookupGood(t *testing.T) {
	ip, err := (ipLookupClient{
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
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
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
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
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
	}).doWithCustomFunc(ctx, invalidIPLookup)
	if !errors.Is(err, ErrInvalidIPAddress) {
		t.Fatal("expected an error here")
	}
	if ip != DefaultProbeIP {
		t.Fatal("expected the default IP here")
	}
}
