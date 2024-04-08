package oonimkall_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
	"github.com/ooni/probe-cli/v3/pkg/oonimkall"
)

func TestSessionHTTPDo(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
		// Implementation note: because we need to backport this patch to the release/3.18
		// branch, it would be quite verbose and burdensome use netem to implement this test,
		// since release/3.18 is lagging behind from master in terms of netemx.
		const expectedResponseBody = "Hello, World!\r\n"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedResponseBody))
		}))
		defer server.Close()

		req := &oonimkall.HTTPRequest{
			Method: "GET",
			Url:    server.URL,
		}

		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		resp, err := sess.HTTPDo(sess.NewContext(), req)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(expectedResponseBody, resp.Body); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("we handle the case where the URL is invalid", func(t *testing.T) {
		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		req := &oonimkall.HTTPRequest{
			Method: "GET",
			Url:    "\t", // this URL is invalid
		}

		resp, err := sess.HTTPDo(sess.NewContext(), req)
		if !strings.HasSuffix(err.Error(), `invalid control character in URL`) {
			t.Fatal("unexpected error", err)
		}
		if resp != nil {
			t.Fatal("expected nil response")
		}
	})

	t.Run("we handle the case where the response body is not 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()

		req := &oonimkall.HTTPRequest{
			Method: "GET",
			Url:    server.URL,
		}

		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		resp, err := sess.HTTPDo(sess.NewContext(), req)
		if !strings.HasSuffix(err.Error(), "HTTP request failed") {
			t.Fatal("unexpected error", err)
		}
		if resp != nil {
			t.Fatal("expected nil response")
		}
	})

	t.Run("we handle the case where the HTTP round trip fails", func(t *testing.T) {
		// Implementation note: because we need to backport this patch to the release/3.18
		// branch, it would be quite verbose and burdensome use netem to implement this test,
		// since release/3.18 is lagging behind from master in terms of netemx.
		server := testingx.MustNewTLSServer(testingx.TLSHandlerReset())
		defer server.Close()

		URL := &url.URL{
			Scheme: "https",
			Host:   server.Endpoint(),
			Path:   "/",
		}

		req := &oonimkall.HTTPRequest{
			Method: "GET",
			Url:    URL.String(),
		}

		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		resp, err := sess.HTTPDo(sess.NewContext(), req)
		if !strings.HasSuffix(err.Error(), "connection_reset") {
			t.Fatal("unexpected error", err)
		}
		if resp != nil {
			t.Fatal("expected nil response")
		}
	})

	t.Run("we handle the case when reading the response body fails", func(t *testing.T) {
		// Implementation note: because we need to backport this patch to the release/3.18
		// branch, it would be quite verbose and burdensome use netem to implement this test,
		// since release/3.18 is lagging behind from master in terms of netemx.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{"))
			hijacker := w.(http.Hijacker)
			conn, _, err := hijacker.Hijack()
			runtimex.PanicOnError(err, "hijacker.Hijack failed")
			if tc, ok := conn.(*net.TCPConn); ok {
				tc.SetLinger(0)
			}
			conn.Close()
		}))
		defer server.Close()

		req := &oonimkall.HTTPRequest{
			Method: "GET",
			Url:    server.URL,
		}

		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		resp, err := sess.HTTPDo(sess.NewContext(), req)
		if !strings.HasSuffix(err.Error(), "connection_reset") {
			t.Fatal("unexpected error", err)
		}
		if resp != nil {
			t.Fatal("expected nil response")
		}
	})
}
