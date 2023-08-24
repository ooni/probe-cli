package webconnectivityqa

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSBlockingAndroidDNSCacheNoData(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	tc := dnsBlockingAndroidDNSCacheNoData()
	tc.Configure(env)

	env.Do(func() {
		reso := netxlite.NewStdlibResolver(log.Log)
		addrs, err := reso.LookupHost(context.Background(), "www.example.com")
		if !errors.Is(err, netxlite.ErrAndroidDNSCacheNoData) {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) != 0 {
			t.Fatal("expected to see no addresses")
		}
	})
}
