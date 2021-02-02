package webconnectivity_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
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
