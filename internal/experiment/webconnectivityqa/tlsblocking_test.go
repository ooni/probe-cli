package webconnectivityqa

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestBlockingTLSConnectionResetWithConsistentDNS(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	tc := tlsBlockingConnectionResetWithConsistentDNS()
	tc.Configure(env)

	env.Do(func() {
		urls := []string{"https://www.example.com/", "https://www.example.com/"}
		for _, URL := range urls {
			t.Run(fmt.Sprintf("for %s", URL), func(t *testing.T) {
				// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
				client := netxlite.NewHTTPClientStdlib(log.Log)
				req, err := http.NewRequest("GET", URL, nil)
				if err != nil {
					t.Fatal(err)
				}
				resp, err := client.Do(req)
				if err == nil || err.Error() != netxlite.FailureConnectionReset {
					t.Fatal("unexpected error", err)
				}
				if resp != nil {
					t.Fatal("expected nil request")
				}
			})
		}
	})
}

func TestBlockingTLSConnectionResetWithInconsistentDNS(t *testing.T) {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	tc := tlsBlockingConnectionResetWithInconsistentDNS()
	tc.Configure(env)

	env.Do(func() {
		urls := []string{"https://www.example.com/", "https://www.example.com/"}
		for _, URL := range urls {
			t.Run(fmt.Sprintf("for %s", URL), func(t *testing.T) {
				// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
				client := netxlite.NewHTTPClientStdlib(log.Log)
				req, err := http.NewRequest("GET", URL, nil)
				if err != nil {
					t.Fatal(err)
				}
				resp, err := client.Do(req)
				if err == nil || err.Error() != netxlite.FailureConnectionReset {
					t.Fatal("unexpected error", err)
				}
				if resp != nil {
					t.Fatal("expected nil request")
				}
			})
		}

		t.Run("there is DNS injection", func(t *testing.T) {
			expect := []string{netemx.ISPProxyAddress}

			t.Run("with stdlib resolver", func(t *testing.T) {
				netx := &netxlite.Netx{}
				reso := netx.NewStdlibResolver(log.Log)
				addrs, err := reso.LookupHost(context.Background(), "www.example.com")
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(expect, addrs); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("with UDP resolver", func(t *testing.T) {
				netx := &netxlite.Netx{}
				d := netx.NewDialerWithoutResolver(log.Log)
				reso := netx.NewParallelUDPResolver(log.Log, d, "8.8.8.8:53")
				addrs, err := reso.LookupHost(context.Background(), "www.example.com")
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(expect, addrs); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	})
}
