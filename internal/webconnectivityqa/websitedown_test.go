package webconnectivityqa

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestWebsiteDownNoAddrs(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	tc := websiteDownNoAddrs()
	tc.Configure(env)

	netx := &netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack}}

	t.Run("for system resolver", func(t *testing.T) {
		reso := netx.NewStdlibResolver(log.Log)
		addrs, err := reso.LookupHost(context.Background(), "www.example.com")
		if err == nil || err.Error() != netxlite.FailureDNSNoAnswer {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) > 0 {
			t.Fatal("expected empty addrs")
		}
	})

	t.Run("for UDP resolver", func(t *testing.T) {
		d := netx.NewDialerWithoutResolver(log.Log)
		reso := netx.NewParallelUDPResolver(log.Log, d, "8.8.8.8:53")
		addrs, err := reso.LookupHost(context.Background(), "www.example.com")
		if err == nil || err.Error() != netxlite.FailureDNSNoAnswer {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) > 0 {
			t.Fatal("expected empty addrs")
		}
	})
}
