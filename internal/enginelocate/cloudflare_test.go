package enginelocate

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// cloudflareRealisticresponse is a realistic response returned by cloudflare
// with the IP address modified to belong to a public institution.
var cloudflareRealisticResponse = []byte(`
fl=270f47
h=www.cloudflare.com
ip=130.192.91.211
ts=1713946961.154
visit_scheme=https
uag=Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:125.0) Gecko/20100101 Firefox/125.0
colo=MXP
sliver=none
http=http/3
loc=IT
tls=TLSv1.3
sni=plaintext
warp=off
gateway=off
rbi=off
kex=X25519
`)

func TestIPLookupWorksUsingcloudlflare(t *testing.T) {

	// We want to make sure the real server gives us an IP address.
	t.Run("is working as intended when using the real server", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// figure out the IP address using cloudflare
		netx := &netxlite.Netx{}
		ip, err := cloudflareIPLookup(
			context.Background(),
			http.DefaultClient,
			log.Log,
			model.HTTPHeaderUserAgent,
			netx.NewStdlibResolver(model.DiscardLogger),
		)

		// we expect this call to succeed
		if err != nil {
			t.Fatal(err)
		}

		// we expect to get back a valid IPv4/IPv6 address
		if net.ParseIP(ip) == nil {
			t.Fatalf("not an IP address: '%s'", ip)
		}
	})

	// But we also want to make sure everything is working as intended when using
	// a local HTTP server, as well as that we can handle errors, so that we can run
	// tests in short mode. This is done with the tests below.

	t.Run("is working as intended when using a fake server", func(t *testing.T) {
		// create a fake server returning an hardcoded IP address.
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(cloudflareRealisticResponse)
		}))
		defer srv.Close()

		// create an HTTP client that uses the fake server.
		client := &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				// rewrite the request URL to be the one of the fake server
				req.URL = runtimex.Try1(url.Parse(srv.URL))
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// figure out the IP address using cloudflare
		netx := &netxlite.Netx{}
		ip, err := cloudflareIPLookup(
			context.Background(),
			client,
			log.Log,
			model.HTTPHeaderUserAgent,
			netx.NewStdlibResolver(model.DiscardLogger),
		)

		// we expect this call to succeed
		if err != nil {
			t.Fatal(err)
		}

		// we expect to get back a valid IPv4/IPv6 address
		if net.ParseIP(ip) == nil {
			t.Fatalf("not an IP address: '%s'", ip)
		}

		// we expect to see exactly the IP address that we want to see
		if ip != "130.192.91.211" {
			t.Fatal("unexpected IP address", ip)
		}
	})

	t.Run("correctly handles errors", func(t *testing.T) {
		// create a fake server resetting the connection for the client.
		srv := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer srv.Close()

		// create an HTTP client that uses the fake server.
		client := &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				// rewrite the request URL to be the one of the fake server
				req.URL = runtimex.Try1(url.Parse(srv.URL))
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// figure out the IP address using cloudflare
		netx := &netxlite.Netx{}
		ip, err := cloudflareIPLookup(
			context.Background(),
			client,
			log.Log,
			model.HTTPHeaderUserAgent,
			netx.NewStdlibResolver(model.DiscardLogger),
		)

		// we expect to see ECONNRESET here
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// the returned IP address should be the default one
		if ip != model.DefaultProbeIP {
			t.Fatal("unexpected IP address", ip)
		}
	})

	t.Run("correctly handles the case where there's no IP addreess", func(t *testing.T) {
		// create a fake server returnning different keys
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`ipx=130.192.91.211`)) // note: different key name
		}))
		defer srv.Close()

		// create an HTTP client that uses the fake server.
		client := &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				// rewrite the request URL to be the one of the fake server
				req.URL = runtimex.Try1(url.Parse(srv.URL))
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// figure out the IP address using cloudflare
		netx := &netxlite.Netx{}
		ip, err := cloudflareIPLookup(
			context.Background(),
			client,
			log.Log,
			model.HTTPHeaderUserAgent,
			netx.NewStdlibResolver(model.DiscardLogger),
		)

		// we expect to see ECONNRESET here
		if !errors.Is(err, ErrInvalidIPAddress) {
			t.Fatal("unexpected error", err)
		}

		// the returned IP address should be the default one
		if ip != model.DefaultProbeIP {
			t.Fatal("unexpected IP address", ip)
		}
	})
}
