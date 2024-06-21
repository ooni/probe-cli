package webconnectivityalgo

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// This function tests the [CallWebConnectivityTestHelper] function.
func TestSessionCallWebConnectivityTestHelper(t *testing.T) {
	// We start with simple tests that exercise the basic functionality of the method
	// without bothering with having more than one available test helper.

	t.Run("when there are no available test helpers", func(t *testing.T) {
		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &mocks.Session{
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
			MockDefaultHTTPClient: func() model.HTTPClient {
				return http.DefaultClient
			},
			MockUserAgent: func() string {
				return model.HTTPHeaderUserAgent
			},
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// invoke the API
		cresp, idx, err := CallWebConnectivityTestHelper(ctx, creq, nil, sess)

		// make sure we get the expected error
		if !errors.Is(err, model.ErrNoAvailableTestHelpers) {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is zero
		if idx != 0 {
			t.Fatal("expected zero, got", idx)
		}

		// make sure cresp is nil
		if cresp != nil {
			t.Fatal("expected nil, got", cresp)
		}
	})

	t.Run("when the call fails", func(t *testing.T) {
		// create a local test server that always resets the connection
		server := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer server.Close()

		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &mocks.Session{
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
			MockDefaultHTTPClient: func() model.HTTPClient {
				return http.DefaultClient
			},
			MockUserAgent: func() string {
				return model.HTTPHeaderUserAgent
			},
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// create the list of test helpers to use
		testhelpers := []model.OOAPIService{{
			Address: server.URL,
			Type:    "https",
			Front:   "",
		}}

		// invoke the API
		cresp, idx, err := CallWebConnectivityTestHelper(ctx, creq, testhelpers, sess)

		// make sure we get the expected error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is zero
		if idx != 0 {
			t.Fatal("expected zero, got", idx)
		}

		// make sure cresp is nil
		if cresp != nil {
			t.Fatal("expected nil, got", cresp)
		}
	})

	t.Run("when the call succeeds", func(t *testing.T) {
		// create a local test server that always returns an ~empty response
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &mocks.Session{
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
			MockDefaultHTTPClient: func() model.HTTPClient {
				return http.DefaultClient
			},
			MockUserAgent: func() string {
				return model.HTTPHeaderUserAgent
			},
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// create the list of test helpers to use
		testhelpers := []model.OOAPIService{{
			Address: server.URL,
			Type:    "https",
			Front:   "",
		}}

		// invoke the API
		cresp, idx, err := CallWebConnectivityTestHelper(ctx, creq, testhelpers, sess)

		// make sure we get the expected error
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is zero
		if idx != 0 {
			t.Fatal("expected zero, got", idx)
		}

		// make sure cresp is not nil
		if cresp == nil {
			t.Fatal("expected not nil, got", cresp)
		}
	})

	// Now we include some tests that ensure that we can chain (in more or less
	// smart fashion) API calls using multiple test helper endpoints.

	t.Run("with two test helpers where the first one resets the connection and the second works", func(t *testing.T) {
		// create a local test server1 that always resets the connection
		server1 := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer server1.Close()

		// create a local test server2 that always returns an ~empty response
		server2 := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server2.Close()

		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &mocks.Session{
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
			MockDefaultHTTPClient: func() model.HTTPClient {
				return http.DefaultClient
			},
			MockUserAgent: func() string {
				return model.HTTPHeaderUserAgent
			},
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// create the list of test helpers to use
		testhelpers := []model.OOAPIService{{
			Address: server1.URL,
			Type:    "https",
			Front:   "",
		}, {
			Address: server2.URL,
			Type:    "https",
			Front:   "",
		}}

		// invoke the API
		cresp, idx, err := CallWebConnectivityTestHelper(ctx, creq, testhelpers, sess)

		// make sure we get the expected error
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is one
		if idx != 1 {
			t.Fatal("expected one, got", idx)
		}

		// make sure cresp is not nil
		if cresp == nil {
			t.Fatal("expected not nil, got", cresp)
		}
	})

	t.Run("with two test helpers where the first one times out the connection and the second works", func(t *testing.T) {
		// create a local test server1 that resets the connection after a ~long delay
		server1 := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(10 * time.Second):
				testingx.HTTPHandlerReset().ServeHTTP(w, r)
			case <-r.Context().Done():
				return
			}
		}))
		defer server1.Close()

		// create a local test server2 that always returns an ~empty response
		server2 := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server2.Close()

		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &mocks.Session{
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
			MockDefaultHTTPClient: func() model.HTTPClient {
				return http.DefaultClient
			},
			MockUserAgent: func() string {
				return model.HTTPHeaderUserAgent
			},
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// create the list of test helpers to use
		testhelpers := []model.OOAPIService{{
			Address: server1.URL,
			Type:    "https",
			Front:   "",
		}, {
			Address: server2.URL,
			Type:    "https",
			Front:   "",
		}}

		// invoke the API
		cresp, idx, err := CallWebConnectivityTestHelper(ctx, creq, testhelpers, sess)

		// make sure we get the expected error
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is one
		if idx != 1 {
			t.Fatal("expected one, got", idx)
		}

		// make sure cresp is not nil
		if cresp == nil {
			t.Fatal("expected not nil, got", cresp)
		}
	})
}
