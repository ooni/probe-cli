package geolocate

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
)

func TestIPLookupWorksUsingIPInfo(t *testing.T) {
	ip, err := ipInfoIPLookup(
		context.Background(),
		http.DefaultClient,
		log.Log,
		httpheader.UserAgent(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("not an IP address: '%s'", ip)
	}
}
