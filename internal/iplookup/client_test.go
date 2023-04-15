package iplookup

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// This test ensures that Client.httpDo works as intended.
func TestClient_httpDo(t *testing.T) {
	// testcase is a test case for this test.
	type testcase struct {
		// name is the test case name.
		name string

		// handler is the HTTP server's handler.
		handler http.HandlerFunc

		// expectErr is the expected error.
		expectErr error

		// expectBody is the expected response body.
		expectBody []byte
	}

	// testcases contains all the test cases.
	testcases := []testcase{{
		name: "httpClient.Do failure",
		handler: func(w http.ResponseWriter, r *http.Request) {
			hj, ok := w.(http.Hijacker)
			runtimex.Assert(ok, "cannot hijack")
			conn, _ := runtimex.Try2(hj.Hijack())
			conn.Close()
		},
		expectErr:  io.EOF,
		expectBody: nil,
	}, {
		name: "response body is not 200 Ok",
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		},
		expectErr:  ErrHTTPRequestFailed,
		expectBody: nil,
	}, {
		name: "successful case",
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("deadbeef"))
		},
		expectErr:  nil,
		expectBody: []byte("deadbeef"),
	}}

	// run each test case
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// create the testing server to test with
			srvr := httptest.NewServer(http.HandlerFunc(tc.handler))
			defer srvr.Close()

			// create the HTTP request
			req := runtimex.Try1(http.NewRequest("GET", srvr.URL, nil))

			// create the Client instance
			c := &Client{
				Logger:        model.DiscardLogger,
				Resolver:      netxlite.NewStdlibResolver(model.DiscardLogger),
				TestingHTTPDo: nil,
			}

			// issue the request and get the response body
			data, err := c.httpDo(req, FamilyINET)

			// make sure the error is the expected one
			if !errors.Is(err, tc.expectErr) {
				t.Fatal("unexpected error", err)
			}

			// make sure the response body is the expected one
			if !bytes.Equal(tc.expectBody, data) {
				t.Fatal("expected", tc.expectBody, "got", data)
			}
		})
	}
}
