package webconnectivity_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
)

func TestHTTPGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	ctx := context.Background()
	r := webconnectivity.HTTPGet(ctx, webconnectivity.HTTPGetConfig{
		Addresses: []string{"104.16.249.249", "104.16.248.249"},
		Session:   newsession(t, false),
		TargetURL: &url.URL{Scheme: "https", Host: "cloudflare-dns.com", Path: "/"},
	})
	if r.TestKeys.Failure != nil {
		t.Fatal(*r.TestKeys.Failure)
	}
	if r.Failure != nil {
		t.Fatal(*r.Failure)
	}
}

func TestHTTPGetMakeDNSCache(t *testing.T) {
	// test for input being an IP
	out := webconnectivity.HTTPGetMakeDNSCache(
		"1.1.1.1", "1.1.1.1",
	)
	if out != "" {
		t.Fatal("expected empty output here")
	}
	// test for input being a domain
	out = webconnectivity.HTTPGetMakeDNSCache(
		"dns.google", "8.8.8.8 8.8.4.4",
	)
	if out != "dns.google 8.8.8.8 8.8.4.4" {
		t.Fatal("expected ordinary output here")
	}
}
