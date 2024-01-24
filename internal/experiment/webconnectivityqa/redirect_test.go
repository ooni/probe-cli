package webconnectivityqa

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

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
			defer env.Close()

			tc.Configure(env)

			env.Do(func() {
				ports := []string{"80", "443"}

				for _, port := range ports {
					t.Run(fmt.Sprintf("for port %s", port), func(t *testing.T) {
						netx := &netxlite.Netx{}
						dialer := netx.NewDialerWithoutResolver(log.Log)
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
			defer env.Close()

			tc.Configure(env)

			env.Do(func() {
				urls := []string{"http://www.example.com/", "https://www.example.com/"}

				for _, URL := range urls {
					t.Run(fmt.Sprintf("for URL %s", URL), func(t *testing.T) {
						// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
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
			defer env.Close()

			tc.Configure(env)

			env.Do(func() {
				t.Run("with stdlib resolver", func(t *testing.T) {
					netx := &netxlite.Netx{}
					reso := netx.NewStdlibResolver(log.Log)
					addrs, err := reso.LookupHost(context.Background(), "www.example.com")
					if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
						t.Fatal("unexpected error", err)
					}
					if len(addrs) != 0 {
						t.Fatal("expected zero length addrs")
					}
				})

				t.Run("with UDP resolver", func(t *testing.T) {
					netx := &netxlite.Netx{}
					d := netx.NewDialerWithoutResolver(log.Log)
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

func TestRedirectWithConsistentDNSAndThenEOF(t *testing.T) {
	testcases := []*TestCase{
		redirectWithConsistentDNSAndThenEOFForHTTP(),
		redirectWithConsistentDNSAndThenEOFForHTTPS(),
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			tc.Configure(env)

			env.Do(func() {
				urls := []string{"http://www.example.com/", "https://www.example.com/"}

				for _, URL := range urls {
					t.Run(fmt.Sprintf("for URL %s", URL), func(t *testing.T) {
						// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
						client := netxlite.NewHTTPClientStdlib(log.Log)
						req := runtimex.Try1(http.NewRequest("GET", URL, nil))
						resp, err := client.Do(req)
						if err == nil || err.Error() != netxlite.FailureEOFError {
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

func TestRedirectWithConsistentDNSAndThenTimeout(t *testing.T) {
	testcases := []*TestCase{
		redirectWithConsistentDNSAndThenTimeoutForHTTP(),
		redirectWithConsistentDNSAndThenTimeoutForHTTPS(),
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			tc.Configure(env)

			env.Do(func() {
				urls := []string{"http://www.example.com/", "https://www.example.com/"}

				for _, URL := range urls {
					t.Run(fmt.Sprintf("for URL %s", URL), func(t *testing.T) {
						ctx, cancel := context.WithTimeout(context.Background(), time.Second)
						defer cancel()
						// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
						client := netxlite.NewHTTPClientStdlib(log.Log)
						req := runtimex.Try1(http.NewRequestWithContext(ctx, "GET", URL, nil))
						resp, err := client.Do(req)
						if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
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
