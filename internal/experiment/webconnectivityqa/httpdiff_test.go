package webconnectivityqa

import (
	"context"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestHTTPDiffWithConsistentDNS(t *testing.T) {
	testcases := []*TestCase{
		httpDiffWithConsistentDNS(),
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			tc.Configure(env)

			env.Do(func() {
				client := netxlite.NewHTTPClientStdlib(log.Log)
				req := runtimex.Try1(http.NewRequest("GET", "http://www.example.com/", nil))
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff([]byte(netemx.Blockpage), body); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	}
}

func TestHTTPDiffWithInconsistentDNS(t *testing.T) {
	testcases := []*TestCase{
		httpDiffWithInconsistentDNS(),
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			tc.Configure(env)

			env.Do(func() {
				t.Run("there is blockpage spoofing", func(t *testing.T) {
					client := netxlite.NewHTTPClientStdlib(log.Log)
					req := runtimex.Try1(http.NewRequest("GET", "http://www.example.com/", nil))
					resp, err := client.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					defer resp.Body.Close()
					body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
					if err != nil {
						t.Fatal(err)
					}
					if diff := cmp.Diff([]byte(netemx.Blockpage), body); diff != "" {
						t.Fatal(diff)
					}
				})

				t.Run("there is DNS spoofing", func(t *testing.T) {
					expect := []string{netemx.ISPProxyAddress}

					t.Run("with stdlib resolver", func(t *testing.T) {
						reso := netxlite.NewStdlibResolver(log.Log)
						addrs, err := reso.LookupHost(context.Background(), "www.example.com")
						if err != nil {
							t.Fatal(err)
						}
						if diff := cmp.Diff(expect, addrs); diff != "" {
							t.Fatal(diff)
						}
					})

					t.Run("with UDP resolver", func(t *testing.T) {
						d := netxlite.NewDialerWithoutResolver(log.Log)
						reso := netxlite.NewParallelUDPResolver(log.Log, d, "8.8.8.8:53")
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
		})
	}
}
