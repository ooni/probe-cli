package probeservices

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestGetTestHelpers(t *testing.T) {

	// First, let's check whether we can get a response from the real OONI backend.
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// create client
		client := newclient()

		// issue the request
		testhelpers, err := client.GetTestHelpers(context.Background())

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// we expect at least one TH
		if len(testhelpers) <= 1 {
			t.Fatal("no returned test helpers?!")
		}
	})

	// Now let's construct a test server that returns a valid response and try
	// to communicate with such a test server successfully and with errors

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// this is what we expect to receive
		expect := map[string][]model.OOAPIService{
			"web-connectivity": {{
				Address: "https://0.th.ooni.org/",
				Type:    "https",
			}},
		}

		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Method == http.MethodGet, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/test-helpers", "invalid URL path")
			w.Write(must.MarshalJSON(expect))
		}))
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// issue the GET request
		testhelpers, err := client.GetTestHelpers(context.Background())

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect to see exactly what the server sent
		if diff := cmp.Diff(expect, testhelpers); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("reports an error when the connection is reset", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// issue the GET request
		testhelpers, err := client.GetTestHelpers(context.Background())

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// we expect to see a zero-length / nil map
		if len(testhelpers) != 0 {
			t.Fatal("expected result lenght to be zero")
		}
	})

	t.Run("reports an error when the response is not JSON parsable", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{`))
		}))
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// issue the GET request
		testhelpers, err := client.GetTestHelpers(context.Background())

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// we expect to see a zero-length / nil map
		if len(testhelpers) != 0 {
			t.Fatal("expected result lenght to be zero")
		}
	})

	t.Run("correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// issue the GET request
		testhelpers, err := client.GetTestHelpers(context.Background())

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// we expect to see a zero-length / nil map
		if len(testhelpers) != 0 {
			t.Fatal("expected result lenght to be zero")
		}
	})
}
