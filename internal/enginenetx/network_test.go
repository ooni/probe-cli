package enginenetx_test

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/enginenetx"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingsocks5"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNetworkQA(t *testing.T) {
	t.Run("is WAI when not using any proxy", func(t *testing.T) {
		env := netemx.MustNewScenario(netemx.InternetScenario)
		defer env.Close()

		env.Do(func() {
			txp := enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				log.Log,
				nil,
				netxlite.NewStdlibResolver(log.Log),
			)
			client := txp.NewHTTPClient()
			resp, err := client.Get("https://www.example.com/")
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%+v", resp)
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatal("unexpected status code")
			}
		})
	})

	t.Run("is WAI when using a SOCKS5 proxy", func(t *testing.T) {
		// create internet measurement scenario
		env := netemx.MustNewScenario(netemx.InternetScenario)
		defer env.Close()

		// create a proxy using the client's TCP/IP stack
		proxy := testingsocks5.MustNewServer(
			log.Log,
			&netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack}},
			&net.TCPAddr{
				IP:   net.ParseIP(env.ClientStack.IPAddress()),
				Port: 9050,
			},
		)
		defer proxy.Close()

		env.Do(func() {
			txp := enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				log.Log,
				&url.URL{
					Scheme: "socks5",
					Host:   net.JoinHostPort(env.ClientStack.IPAddress(), "9050"),
					Path:   "/",
				},
				netxlite.NewStdlibResolver(log.Log),
			)
			client := txp.NewHTTPClient()

			// To make sure we're connecting to the expected endpoint, we're going to use
			// measurexlite and tracing to observe the destination endpoints
			trace := measurexlite.NewTrace(0, time.Now())
			ctx := netxlite.ContextWithTrace(context.Background(), trace)

			// create request using the above context
			//
			// Implementation note: we cannot use HTTPS with netem here as explained
			// by the https://github.com/ooni/probe/issues/2536 issue.
			req, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%+v", resp)
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatal("unexpected status code")
			}

			// make sure that we only connected to the SOCKS5 proxy
			tcpConnects := trace.TCPConnects()
			if len(tcpConnects) <= 0 {
				t.Fatal("expected at least one TCP connect")
			}
			for idx, entry := range tcpConnects {
				t.Logf("%d: %+v", idx, entry)
				if entry.IP != env.ClientStack.IPAddress() {
					t.Fatal("unexpected IP address")
				}
				if entry.Port != 9050 {
					t.Fatal("unexpected port")
				}
			}
		})
	})

	t.Run("is WAI when using an HTTP proxy", func(t *testing.T) {
		// create internet measurement scenario
		env := netemx.MustNewScenario(netemx.InternetScenario)
		defer env.Close()

		// create a proxy using the client's TCP/IP stack
		proxy := testingx.MustNewHTTPServerEx(
			&net.TCPAddr{IP: net.ParseIP(env.ClientStack.IPAddress()), Port: 8080},
			env.ClientStack,
			testingx.NewHTTPProxyHandler(log.Log, &netxlite.Netx{
				Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack}}),
		)
		defer proxy.Close()

		env.Do(func() {
			txp := enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				log.Log,
				&url.URL{
					Scheme: "http",
					Host:   net.JoinHostPort(env.ClientStack.IPAddress(), "8080"),
					Path:   "/",
				},
				netxlite.NewStdlibResolver(log.Log),
			)
			client := txp.NewHTTPClient()

			// To make sure we're connecting to the expected endpoint, we're going to use
			// measurexlite and tracing to observe the destination endpoints
			trace := measurexlite.NewTrace(0, time.Now())
			ctx := netxlite.ContextWithTrace(context.Background(), trace)

			// create request using the above context
			//
			// Implementation note: we cannot use HTTPS with netem here as explained
			// by the https://github.com/ooni/probe/issues/2536 issue.
			req, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%+v", resp)
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatal("unexpected status code")
			}

			// make sure that we only connected to the HTTP proxy
			tcpConnects := trace.TCPConnects()
			if len(tcpConnects) <= 0 {
				t.Fatal("expected at least one TCP connect")
			}
			for idx, entry := range tcpConnects {
				t.Logf("%d: %+v", idx, entry)
				if entry.IP != env.ClientStack.IPAddress() {
					t.Fatal("unexpected IP address")
				}
				if entry.Port != 8080 {
					t.Fatal("unexpected port")
				}
			}
		})
	})

	t.Run("is WAI when using an HTTPS proxy", func(t *testing.T) {
		// create internet measurement scenario
		env := netemx.MustNewScenario(netemx.InternetScenario)
		defer env.Close()

		// create a proxy using the client's TCP/IP stack
		proxy := testingx.MustNewHTTPServerTLSEx(
			&net.TCPAddr{IP: net.ParseIP(env.ClientStack.IPAddress()), Port: 4443},
			env.ClientStack,
			testingx.NewHTTPProxyHandler(log.Log, &netxlite.Netx{
				Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack}}),
			env.ClientStack,
			"proxy.local",
		)
		defer proxy.Close()

		env.Do(func() {
			txp := enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				log.Log,
				&url.URL{
					Scheme: "https",
					Host:   net.JoinHostPort(env.ClientStack.IPAddress(), "4443"),
					Path:   "/",
				},
				netxlite.NewStdlibResolver(log.Log),
			)
			client := txp.NewHTTPClient()

			// To make sure we're connecting to the expected endpoint, we're going to use
			// measurexlite and tracing to observe the destination endpoints
			trace := measurexlite.NewTrace(0, time.Now())
			ctx := netxlite.ContextWithTrace(context.Background(), trace)

			// create request using the above context
			//
			// Implementation note: we cannot use HTTPS with netem here as explained
			// by the https://github.com/ooni/probe/issues/2536 issue.
			req, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%+v", resp)
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatal("unexpected status code")
			}

			// make sure that we only connected to the HTTPS proxy
			tcpConnects := trace.TCPConnects()
			if len(tcpConnects) <= 0 {
				t.Fatal("expected at least one TCP connect")
			}
			for idx, entry := range tcpConnects {
				t.Logf("%d: %+v", idx, entry)
				if entry.IP != env.ClientStack.IPAddress() {
					t.Fatal("unexpected IP address")
				}
				if entry.Port != 4443 {
					t.Fatal("unexpected port")
				}
			}
		})
	})

	t.Run("NewHTTPClient returns a client with a cookie jar", func(t *testing.T) {
		txp := enginenetx.NewNetwork(
			bytecounter.New(),
			&kvstore.Memory{},
			log.Log,
			nil,
			netxlite.NewStdlibResolver(log.Log),
		)
		client := txp.NewHTTPClient()
		if client.Jar == nil {
			t.Fatal("expected non-nil cookie jar")
		}
	})
}
