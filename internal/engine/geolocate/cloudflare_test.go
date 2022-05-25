package geolocate

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestIPLookupWorksUsingcloudlflare(t *testing.T) {
	ip, err := cloudflareIPLookup(
		context.Background(),
		http.DefaultClient,
		log.Log,
		model.HTTPHeaderUserAgent,
	)
	if err != nil {
		t.Fatal(err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("not an IP address: '%s'", ip)
	}
}
