package probeservices

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestMaybeLogin(t *testing.T) {
	// First, let's check whether we can get a response from the real OONI backend.
	t.Run("is working as intended with the real OONI backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// create client
		clnt := newclient()

		// we need to register first because we don't have state yet
		if err := clnt.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// now we try to login and get a token
		if err := clnt.MaybeLogin(context.Background()); err != nil {
			t.Fatal(err)
		}

		// do this again, and later on we'll verify that we
		// did actually issue just a single login call
		if err := clnt.MaybeLogin(context.Background()); err != nil {
			t.Fatal(err)
		}

		// make sure we did call login just once: the second call
		// should not invoke login because we have good state
		if clnt.LoginCalls.Load() != 1 {
			t.Fatal("called login API too many times")
		}
	})

	// Now let's construct a test server that returns a valid response and try
	// to communicate with such a test server successfully and with errors

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONIBackendWithLoginFlow{}

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

		// we need to register first because we don't have state yet
		if err := client.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// now we try to login and get a token
		if err := client.MaybeLogin(context.Background()); err != nil {
			t.Fatal(err)
		}

		// do this again, and later on we'll verify that we
		// did actually issue just a single login call
		if err := client.MaybeLogin(context.Background()); err != nil {
			t.Fatal(err)
		}

		// make sure we did call login just once: the second call
		// should not invoke login because we have good state
		if client.LoginCalls.Load() != 1 {
			t.Fatal("called login API too many times")
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

		// we need to convince the client that we're registered first otherwise it will
		// refuse to send a request to the server and we won't be testing networking
		runtimex.Try0(client.StateFile.Set(State{
			ClientID: "ttt-uuu-iii",
			Expire:   time.Time{}, // explicitly empty
			Password: "xxx-xxx-xxx",
			Token:    "", // explicitly empty
		}))

		// now we try to login and get a token
		err := client.MaybeLogin(context.Background())

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// make sure we did call login
		if client.LoginCalls.Load() != 1 {
			t.Fatal("called login API the wrong number of times")
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

		// we need to convince the client that we're registered first otherwise it will
		// refuse to send a request to the server and we won't be testing networking
		runtimex.Try0(client.StateFile.Set(State{
			ClientID: "ttt-uuu-iii",
			Expire:   time.Time{}, // explicitly empty
			Password: "xxx-xxx-xxx",
			Token:    "", // explicitly empty
		}))

		// now we try to login and get a token
		err := client.MaybeLogin(context.Background())

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// make sure we did call login
		if client.LoginCalls.Load() != 1 {
			t.Fatal("called login API the wrong number of times")
		}
	})

	t.Run("when we already have a token", func(t *testing.T) {
		clnt := newclient()

		// create a state with valid expire and token
		state := State{
			Expire: time.Now().Add(time.Hour),
			Token:  "xx-xxx-x-xxxx",
		}

		// synchronize the state
		if err := clnt.StateFile.Set(state); err != nil {
			t.Fatal(err)
		}

		// now call loging and we expect no error because we should
		// already have what we need to perform a login
		if err := clnt.MaybeLogin(context.Background()); err != nil {
			t.Fatal(err)
		}

		// make sure we did not call login
		if clnt.LoginCalls.Load() != 0 {
			t.Fatal("called login API the wrong number of times")
		}
	})

	t.Run("when we have not registered yet", func(t *testing.T) {
		clnt := newclient()

		// With explicitly empty state so it's pretty obvioust there's no state
		state := State{}

		// synchronize the state
		if err := clnt.StateFile.Set(state); err != nil {
			t.Fatal(err)
		}

		// now try to login and expect to see we've not registered yet
		if err := clnt.MaybeLogin(context.Background()); !errors.Is(err, ErrNotRegistered) {
			t.Fatal("unexpected error", err)
		}

		// make sure we did not call login
		if clnt.LoginCalls.Load() != 0 {
			t.Fatal("called login API the wrong number of times")
		}
	})

	t.Run("correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// we need to convince the client that we're registered first otherwise it will
		// refuse to send a request to the server and we won't be testing networking
		runtimex.Try0(client.StateFile.Set(State{
			ClientID: "ttt-uuu-iii",
			Expire:   time.Time{}, // explicitly empty
			Password: "xxx-xxx-xxx",
			Token:    "", // explicitly empty
		}))

		// now we try to login and get a token
		err := client.MaybeLogin(context.Background())

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// make sure we did call login
		if client.LoginCalls.Load() != 1 {
			t.Fatal("called login API the wrong number of times")
		}
	})
}
