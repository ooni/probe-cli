package probeservices

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestFetchTorTargets(t *testing.T) {
	// Implementation note: OONIBackendWithLoginFlow ensures that tor sends a query
	// string and there are also tests making sure of that. We use to have a test that
	// checked for the query string here, but now it seems unnecessary.

	// torflow is the flow with which we invoke the tor API
	torflow := func(t *testing.T, client *Client) (map[string]model.OOAPITorTarget, error) {
		// we need to make sure we're registered and logged in
		if err := client.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}
		if err := client.MaybeLogin(context.Background()); err != nil {
			t.Fatal(err)
		}

		// then we can try to fetch the config
		return client.FetchTorTargets(context.Background(), "ZZ")
	}

	// First, let's check whether we can get a response from the real OONI backend.
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		clnt := newclient()

		// run the tor flow
		targets, err := torflow(t, clnt)

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// we expect non-zero length targets
		if len(targets) <= 0 {
			t.Fatal("expected non-zero-length targets")
		}
	})

	// Now let's construct a test server that returns a valid response and try
	// to communicate with such a test server successfully and with errors

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONIBackendWithLoginFlow{}

		// make sure we return something that is JSON parseable and non-zero-length
		state.SetTorTargets([]byte(`{"foo": {}}`))

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

		// run the tor flow
		targets, err := torflow(t, client)

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// we expect non-zero length targets
		if len(targets) <= 0 {
			t.Fatal("expected non-zero-length targets")
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

		// run the tor flow
		targets, err := torflow(t, client)

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// we expect to see zero-length targets
		if len(targets) != 0 {
			t.Fatal("expected targets to be zero length")
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

		// we need to convince the client that we're logged in first otherwise it will
		// refuse to send a request to the server and we won't be testing networking
		runtimex.Try0(client.StateFile.Set(State{
			ClientID: "ttt-uuu-iii",
			Expire:   time.Now().Add(30 * time.Hour),
			Password: "xxx-xxx-xxx",
			Token:    "abc-yyy-zzz",
		}))

		// run the tor flow
		targets, err := torflow(t, client)

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// we expect to see zero-length targets
		if len(targets) != 0 {
			t.Fatal("expected targets to be zero length")
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

		// run the tor flow
		targets, err := clnt.FetchTorTargets(context.Background(), "ZZ")

		// ensure that the error says we're not registered
		if !errors.Is(err, ErrNotRegistered) {
			t.Fatal("expected an error here")
		}

		// we expect zero length targets
		if len(targets) != 0 {
			t.Fatal("expected zero-length targets")
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

		targets, err := client.FetchTorTargets(context.Background(), "ZZ")

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// we expect zero length targets
		if len(targets) != 0 {
			t.Fatal("expected zero-length targets")
		}
	})

	t.Run("is not logging the response body", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONIBackendWithLoginFlow{}

		// make sure we return something that is JSON parseable
		state.SetTorTargets([]byte(`{}`))

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

		// then we can try to fetch the targets
		targets, err := torflow(t, client)

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// we expect to see zero-length targets
		if len(targets) != 0 {
			t.Fatal("expected targets to be zero length")
		}

		// assert that there are no logs
		//
		// the register, login, and tor API should not log their bodies
		// especially the tor response body and, for backwards consistency,
		// also the other APIs should not emit logs
		if diff := cmp.Diff([]string{}, logger.AllLines()); diff != "" {
			t.Fatal(diff)
		}
	})
}
