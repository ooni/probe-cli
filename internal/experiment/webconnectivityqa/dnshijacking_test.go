package webconnectivityqa

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSHijackingTestCases(t *testing.T) {
	testcases := []*TestCase{
		dnsHijackingToProxyWithHTTPURL(),
		dnsHijackingToProxyWithHTTPSURL(),
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			tc.Configure(env)

			env.Do(func() {
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
}
