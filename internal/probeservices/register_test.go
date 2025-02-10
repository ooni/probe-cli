package probeservices

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestMaybeRegister(t *testing.T) {
	t.Run("is working as intended with the real OONI backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// create client
		clnt := newclient()

		// attempt to register once
		if err := clnt.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// try again (we want to make sure it's idempotent once we've registered)
		if err := clnt.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// make sure we indeed only called it once
		if clnt.RegisterCalls.Load() != 1 {
			t.Fatal("called register API too many times")
		}
	})

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

		// attempt to register once
		if err := client.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// try again (we want to make sure it's idempotent once we've registered)
		if err := client.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// make sure we indeed only called it once
		if client.RegisterCalls.Load() != 1 {
			t.Fatal("called register API too many times")
		}
	})

	t.Run("we can use cloudfronting", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONIBackendWithLoginFlow{}
		mux := state.NewMux()

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

		// attempt to register once
		if err := client.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// try again (we want to make sure it's idempotent once we've registered)
		if err := client.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// make sure we indeed only called it once
		if client.RegisterCalls.Load() != 1 {
			t.Fatal("called register API too many times")
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

		// attempt to register
		err := client.MaybeRegister(context.Background(), MetadataFixture())

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// make sure we did call register
		if client.RegisterCalls.Load() != 1 {
			t.Fatal("called register API the wrong number of times")
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

		// attempt to register
		err := client.MaybeRegister(context.Background(), MetadataFixture())

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// make sure we did call register
		if client.RegisterCalls.Load() != 1 {
			t.Fatal("called register API the wrong number of times")
		}
	})

	t.Run("when metadata is not valid", func(t *testing.T) {
		// we expect ErrInvalidMetadata when metadata is empty
		clnt := newclient()
		err := clnt.MaybeRegister(context.Background(), model.OOAPIProbeMetadata{})
		if !errors.Is(err, ErrInvalidMetadata) {
			t.Fatal("expected an error here")
		}
	})

	t.Run("when we have already registered", func(t *testing.T) {
		clnt := newclient()

		// create a state with valid credentials
		state := State{
			ClientID: "xx-xxx-x-xxxx",
			Password: "xx",
		}

		// synchronize the state
		if err := clnt.StateFile.Set(state); err != nil {
			t.Fatal(err)
		}

		// attempt to register, which should immediately succeed
		if err := clnt.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
			t.Fatal(err)
		}

		// make sure we did not call register
		if clnt.RegisterCalls.Load() != 0 {
			t.Fatal("called register API the wrong number of times")
		}
	})

	t.Run("when the API call fails", func(t *testing.T) {
		clnt := newclient()
		clnt.BaseURL = "\t\t\t" // makes it fail
		ctx := context.Background()
		metadata := MetadataFixture()
		err := clnt.MaybeRegister(ctx, metadata)
		if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
			t.Fatal("expected an error here")
		}
	})

	t.Run("correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// attempt to register
		err := client.MaybeRegister(context.Background(), MetadataFixture())

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// make sure we did call register
		if client.RegisterCalls.Load() != 1 {
			t.Fatal("called register API the wrong number of times")
		}
	})
}
