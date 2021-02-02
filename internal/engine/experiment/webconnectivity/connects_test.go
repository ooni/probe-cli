package webconnectivity_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
)

func TestConnectsSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	ctx := context.Background()
	r := webconnectivity.Connects(ctx, webconnectivity.ConnectsConfig{
		Session:   newsession(t, false),
		TargetURL: &url.URL{Scheme: "https", Host: "cloudflare-dns.com", Path: "/"},
		URLGetterURLs: []string{
			"tlshandshake://104.16.249.249:443", "tlshandshake://104.16.248.249:443",
			"tlshandshake://[2606:4700::6810:f9f9]:443",
			"tlshandshake://[2606:4700::6810:f8f9]:443",
		},
	})
	if len(r.AllKeys) != 4 {
		t.Fatal("unexpected number of TestKeys lists")
	}
	if r.Successes < 1 {
		t.Fatal("no successes?!")
	}
	if r.Total != 4 {
		t.Fatal("unexpected number of attempts")
	}
}

func TestConnectsNoInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	ctx := context.Background()
	r := webconnectivity.Connects(ctx, webconnectivity.ConnectsConfig{
		Session:       newsession(t, false),
		TargetURL:     &url.URL{Scheme: "https", Host: "cloudflare-dns.com", Path: "/"},
		URLGetterURLs: []string{},
	})
	if len(r.AllKeys) != 0 {
		t.Fatal("unexpected number of TestKeys lists")
	}
	if r.Successes != 0 {
		t.Fatal("successes?!")
	}
	if r.Total != 0 {
		t.Fatal("unexpected number of attempts")
	}
}
