package geolocate

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
)

func TestIPLookupWorksUsingIPConfig(t *testing.T) {
	if os.Getenv("CI") == "true" {
		// See https://github.com/ooni/probe-cli/pull/259/checks?check_run_id=2166066881#step:5:123
		// as well as https://github.com/ooni/probe/issues/1418.
		t.Skip("This test does not work with GitHub Actions")
	}
	ip, err := ipConfigIPLookup(
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
