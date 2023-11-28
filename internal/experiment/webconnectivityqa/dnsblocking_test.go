package webconnectivityqa

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSBlockingAndroidDNSCacheNoData(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

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

func TestDNSBlockingNXDOMAIN(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	tc := dnsBlockingNXDOMAIN()
	tc.Configure(env)

	env.Do(func() {
		reso := netxlite.NewStdlibResolver(log.Log)
		addrs, err := reso.LookupHost(context.Background(), "www.example.com")
		if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) != 0 {
			t.Fatal("expected to see no addresses")
		}
	})
}

func TestDNSBlockingBOGON(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	tc := dnsBlockingBOGON()
	tc.Configure(env)

	env.Do(func() {
		reso := netxlite.NewStdlibResolver(log.Log)
		addrs, err := reso.LookupHost(context.Background(), "www.example.com")
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"10.10.34.35"}, addrs); diff != "" {
			t.Fatal(diff)
		}
	})
}
