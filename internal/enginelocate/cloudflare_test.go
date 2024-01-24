package enginelocate

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestIPLookupWorksUsingcloudlflare(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	netx := &netxlite.Netx{}
	ip, err := cloudflareIPLookup(
		context.Background(),
		http.DefaultClient,
		log.Log,
		model.HTTPHeaderUserAgent,
		netx.NewStdlibResolver(model.DiscardLogger),
	)
	if err != nil {
		t.Fatal(err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("not an IP address: '%s'", ip)
	}
}
