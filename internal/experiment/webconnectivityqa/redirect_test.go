package webconnectivityqa

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestRedirectWithConsistentDNSAndThenConnectionRefused(t *testing.T) {
	testcases := []*TestCase{
		redirectWithConsistentDNSAndThenConnectionRefusedForHTTP(),
		redirectWithConsistentDNSAndThenConnectionRefusedForHTTPS(),
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			tc.Configure(env)

			env.Do(func() {
				ports := []string{"80", "443"}

				for _, port := range ports {
					t.Run(fmt.Sprintf("for port %s", port), func(t *testing.T) {
						dialer := netxlite.NewDialerWithoutResolver(log.Log)
						endpoint := net.JoinHostPort(netemx.AddressWwwExampleCom, port)
						conn, err := dialer.DialContext(context.Background(), "tcp", endpoint)
						if err == nil || err.Error() != netxlite.FailureConnectionRefused {
							t.Fatal("unexpected err", err)
						}
						if conn != nil {
							t.Fatal("expected nil conn")
						}
					})
				}
			})
		})
	}
}

func TestRedirectWithConsistentDNSAndThenConnectionReset(t *testing.T) {
	testcases := []*TestCase{
		redirectWithConsistentDNSAndThenConnectionResetForHTTP(),
		redirectWithConsistentDNSAndThenConnectionResetForHTTPS(),
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			tc.Configure(env)

			env.Do(func() {
				urls := []string{"http://www.example.com/", "https://www.example.com/"}

				for _, URL := range urls {
					t.Run(fmt.Sprintf("for URL %s", URL), func(t *testing.T) {
						client := netxlite.NewHTTPClientStdlib(log.Log)
						req := runtimex.Try1(http.NewRequest("GET", URL, nil))
						resp, err := client.Do(req)
						if err == nil || err.Error() != netxlite.FailureConnectionReset {
							t.Fatal("unexpected err", err)
						}
						if resp != nil {
							t.Fatal("expected nil resp")
						}
					})
				}
			})
		})
	}
}

func TestRedirectWithConsistentDNSAndThenNXDOMAIN(t *testing.T) {
	testcases := []*TestCase{
		redirectWithConsistentDNSAndThenNXDOMAIN(),
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			tc.Configure(env)

			env.Do(func() {
				t.Run("with stdlib resolver", func(t *testing.T) {
					reso := netxlite.NewStdlibResolver(log.Log)
					addrs, err := reso.LookupHost(context.Background(), "www.example.com")
					if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
						t.Fatal("unexpected error", err)
					}
					if len(addrs) != 0 {
						t.Fatal("expected zero length addrs")
					}
				})

				t.Run("with UDP resolver", func(t *testing.T) {
					d := netxlite.NewDialerWithoutResolver(log.Log)
					reso := netxlite.NewParallelUDPResolver(log.Log, d, "8.8.8.8:53")
					addrs, err := reso.LookupHost(context.Background(), "www.example.com")
					if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
						t.Fatal("unexpected error", err)
					}
					if len(addrs) != 0 {
						t.Fatal("expected zero length addrs")
					}
				})
			})
		})
	}
}
