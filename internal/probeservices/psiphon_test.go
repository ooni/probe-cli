package probeservices

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestFetchPsiphonConfig(t *testing.T) {

	// psiphonflow is the flow with which we invoke the psiphon API
	psiphonflow := func(t *testing.T, client *Client) ([]byte, error) {
		// we need to make sure we're registered and logged in
		if err := client.MaybeRegister(context.Background(), "", MetadataFixture()); err != nil {
			t.Fatal(err)
		}
		if err := client.MaybeLogin(context.Background(), ""); err != nil {
			t.Fatal(err)
		}

		// then we can try to fetch the config
		return client.FetchPsiphonConfig(context.Background())
	}

	// First, let's check whether we can get a response from the real OONI backend.
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		clnt := newclient()

		// run the psiphon flow
		data, err := psiphonflow(t, clnt)

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
		// create state for emulating the OONI backend
		state := &testingx.OONIBackendWithLoginFlow{}

		// make sure we return something that is JSON parseable
		state.SetPsiphonConfig([]byte(`{}`))

		// expose the state via HTTP
		srv := testingx.MustNewHTTPServer(state.NewMux())
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client so we speak with our local server rather than the true backend
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
		data, err := psiphonflow(t, client)

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

	t.Run("we can use cloudfronting", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONIBackendWithLoginFlow{}
		mux := state.NewMux()

		// make sure we return something that is JSON parseable
		state.SetPsiphonConfig([]byte(`{}`))

		// expose the state via HTTP
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Host == "www.cloudfront.com", "invalid r.Host")
			mux.ServeHTTP(w, r)
		}))
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// make sure we're using cloudfronting
		client.Host = "www.cloudfront.com"

		// override the HTTP client so we speak with our local server rather than the true backend
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
		data, err := psiphonflow(t, client)

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

		// we need to convince the client that we're logged in first otherwise it will
		// refuse to send a request to the server and we won't be testing networking
		runtimex.Try0(client.StateFile.Set(State{
			ClientID: "ttt-uuu-iii",
			Expire:   time.Now().Add(30 * time.Hour),
			Password: "xxx-xxx-xxx",
			Token:    "abc-yyy-zzz",
		}))

		// issue the call directly: no register or login otherwise we're testing
		// the register or login implementation LOL
		data, err := client.FetchPsiphonConfig(context.Background())

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// we expect to see zero-length data
		if len(data) != 0 {
			t.Fatal("expected result lenght to be zero")
		}
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

	t.Run("correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// we need to convince the client that we're logged in first otherwise it will
		// refuse to send a request to the server and we won't be testing networking
		runtimex.Try0(client.StateFile.Set(State{
			ClientID: "ttt-uuu-iii",
			Expire:   time.Now().Add(30 * time.Hour),
			Password: "xxx-xxx-xxx",
			Token:    "abc-yyy-zzz",
		}))

		// issue the API call proper
		data, err := client.FetchPsiphonConfig(context.Background())

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// we expect data to be zero length
		if len(data) != 0 {
			t.Fatal("expected zero length data")
		}
	})

	t.Run("is not logging the response body", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONIBackendWithLoginFlow{}

		// make sure we return something that is JSON parseable
		state.SetPsiphonConfig([]byte(`{}`))

		// expose the state via HTTP
		srv := testingx.MustNewHTTPServer(state.NewMux())
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// create and use a logger for collecting logs
		logger := &testingx.Logger{}
		client.Logger = logger

		// override the HTTP client so we speak with our local server rather than the true backend
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
		data, err := psiphonflow(t, client)

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// the config is bytes but we want to make sure we can parse it
		var config interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatal(err)
		}

		// assert that there are no logs
		//
		// the register, login, and psiphon API should not log their bodies
		if diff := cmp.Diff([]string{}, logger.AllLines()); diff != "" {
			t.Fatal(diff)
		}
	})
}
