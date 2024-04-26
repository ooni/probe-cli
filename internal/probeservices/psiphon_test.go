package probeservices

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestFetchPsiphonConfig(t *testing.T) {

	// First, let's check whether we can get a response from the real OONI backend.
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		clnt := newclient()

		// preconditions: to fetch the psiphon config we need to be register and login
		if err := clnt.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}
		if err := clnt.MaybeLogin(context.Background()); err != nil {
			t.Fatal(err)
		}

		// then we can try to fetch the config
		data, err := clnt.FetchPsiphonConfig(context.Background())

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// the config is bytes but we want to make sure we can parse it
		var config interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatal(err)
		}
	})

	// Now let's construct a test server that returns a valid response and try
	// to communicate with such a test server successfully and with errors

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Method == http.MethodGet, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/test-list/psiphon-config", "invalid URL path")
			w.Write(must.MarshalJSON(`{}`))
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

		// then we can try to fetch the config
		data, err := client.FetchPsiphonConfig(context.Background())

		_, _ = data, err
		t.Fatal("this test is too simplistic")
	})

	t.Run("when we're not registered", func(t *testing.T) {
		clnt := newclient()

		// With explicitly empty state so it's pretty obvioust there's no state
		state := State{}

		// force the state to be empty
		if err := clnt.StateFile.Set(state); err != nil {
			t.Fatal(err)
		}

		// attempt to fetch the config
		data, err := clnt.FetchPsiphonConfig(context.Background())

		// ensure that the error says we're not registered
		if !errors.Is(err, ErrNotRegistered) {
			t.Fatal("expected an error here")
		}

		// obviously the data should be empty as well
		if len(data) != 0 {
			t.Fatal("expected nil data here")
		}
	})
}
