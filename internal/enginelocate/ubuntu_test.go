package enginelocate

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// ubuntuRealisticresponse is a realistic response returned by cloudflare
// with the IP address modified to belong to a public institution.
var ubuntuRealisticresponse = []byte(`
<Response>
<Ip>130.192.91.211</Ip>
<Status>OK</Status>
<CountryCode>IT</CountryCode>
<CountryCode3>ITA</CountryCode3>
<CountryName>Italy</CountryName>
<RegionCode>09</RegionCode>
<RegionName>Lombardia</RegionName>
<City>Sesto San Giovanni</City>
<ZipPostalCode>20099</ZipPostalCode>
<Latitude>45.5349</Latitude>
<Longitude>9.2295</Longitude>
<AreaCode>0</AreaCode>
<TimeZone>Europe/Rome</TimeZone>
</Response>
`)

func TestIPLookupWorksUsingUbuntu(t *testing.T) {

	// We want to make sure the real server gives us an IP address.
	t.Run("is working as intended when using the real server", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		netx := &netxlite.Netx{}
		ip, err := ubuntuIPLookup(
			context.Background(),
			http.DefaultClient,
			log.Log,
			model.HTTPHeaderUserAgent,
			netx.NewStdlibResolver(model.DiscardLogger),
		)
		if err != nil {
			t.Fatal(err)
		}
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
			w.Write(ubuntuRealisticresponse)
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

		// figure out the IP address using ubuntu
		netx := &netxlite.Netx{}
		ip, err := ubuntuIPLookup(
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

	t.Run("correctly handles network errors", func(t *testing.T) {
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

		// figure out the IP address using ubuntu
		netx := &netxlite.Netx{}
		ip, err := ubuntuIPLookup(
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

	t.Run("correctly handles parsing errors", func(t *testing.T) {
		// create a fake server returnning different keys
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<`)) // note: invalid XML
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

		// figure out the IP address using ubuntu
		netx := &netxlite.Netx{}
		ip, err := ubuntuIPLookup(
			context.Background(),
			client,
			log.Log,
			model.HTTPHeaderUserAgent,
			netx.NewStdlibResolver(model.DiscardLogger),
		)

		// we expect to see an XML parsing error here
		if err == nil || !strings.HasPrefix(err.Error(), "XML syntax error") {
			t.Fatalf("not the error we expected: %+v", err)
		}

		// the returned IP address should be the default one
		if ip != model.DefaultProbeIP {
			t.Fatal("unexpected IP address", ip)
		}
	})

	t.Run("correctly handles missing IP address in a valid XML document", func(t *testing.T) {
		// create a fake server returnning different keys
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<Response></Response>`)) // note: missing IP address
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

		// figure out the IP address using ubuntu
		netx := &netxlite.Netx{}
		ip, err := ubuntuIPLookup(
			context.Background(),
			client,
			log.Log,
			model.HTTPHeaderUserAgent,
			netx.NewStdlibResolver(model.DiscardLogger),
		)

		// we expect to see an error indicating there's no IP address in the response
		if !errors.Is(err, ErrInvalidIPAddress) {
			t.Fatal("unexpected error", err)
		}

		// the returned IP address should be the default one
		if ip != model.DefaultProbeIP {
			t.Fatal("unexpected IP address", ip)
		}
	})

	t.Run("correctly handles the case where the IP address is invalid", func(t *testing.T) {
		// create a fake server returnning different keys
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<Response><Ip>foobarbaz</Ip></Response>`)) // note: not an IP address
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

		// figure out the IP address using ubuntu
		netx := &netxlite.Netx{}
		ip, err := ubuntuIPLookup(
			context.Background(),
			client,
			log.Log,
			model.HTTPHeaderUserAgent,
			netx.NewStdlibResolver(model.DiscardLogger),
		)

		// we expect to see an error indicating there's no IP address in the response
		if !errors.Is(err, ErrInvalidIPAddress) {
			t.Fatal("unexpected error", err)
		}

		// the returned IP address should be the default one
		if ip != model.DefaultProbeIP {
			t.Fatal("unexpected IP address", ip)
		}
	})
}
