package oonimkall_test

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestOONIRunFetch(t *testing.T) {
	t.Run("we can fetch a OONI Run link descriptor", func(t *testing.T) {
		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		rawResp, err := sess.OONIRunFetch(sess.NewContext(), 9408643002)
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]any{
			"descriptor_creation_time": "2023-07-18T15:38:21Z",
			"descriptor": map[string]any{
				"author":           "simone@openobservatory.org",
				"description":      "We use this OONI Run descriptor for writing integration tests for ooni/probe-cli/v3/pkg/oonimkall.",
				"description_intl": map[string]any{},
				"icon":             "",
				"name":             "OONIMkAll Integration Testing",
				"name_intl":        map[string]any{},
				"nettests": []any{
					map[string]any{
						"backend_options":           map[string]any{},
						"inputs":                    []any{string("https://www.example.com/")},
						"is_background_run_enabled": false,
						"is_manual_run_enabled":     false,
						"options":                   map[string]any{},
						"test_name":                 "web_connectivity",
					},
				},
				"short_description":      "Integration testing descriptor for ooni/probe-cli/v3/pkg/oonimkall.",
				"short_description_intl": map[string]any{},
			},
			"translation_creation_time": "2023-07-18T15:38:21Z",
			"v":                         1.0,
		}

		var got map[string]any
		runtimex.Try0(json.Unmarshal([]byte(rawResp), &got))
		t.Log(got)

		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("we handle the case where the URL is invalid", func(t *testing.T) {
		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		URL := &url.URL{Host: "\t"} // this URL is invalid

		rawResp, err := sess.OONIRunFetchWithURL(sess.NewContext(), URL)
		if !strings.HasSuffix(err.Error(), `invalid URL escape "%09"`) {
			t.Fatal("unexpected error", err)
		}
		if rawResp != "" {
			t.Fatal("expected empty raw response")
		}
	})

	t.Run("we handle the case where the response body is not 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()

		URL := runtimex.Try1(url.Parse(server.URL))

		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		rawResp, err := sess.OONIRunFetchWithURL(sess.NewContext(), URL)
		if !strings.HasSuffix(err.Error(), "HTTP request failed") {
			t.Fatal("unexpected error", err)
		}
		if rawResp != "" {
			t.Fatal("expected empty raw response")
		}
	})

	t.Run("we handle the case where the HTTP round trip fails", func(t *testing.T) {
		// Implementation note: because we need to backport this patch to the release/3.18
		// branch, it would be quite verbose and burdensome use netem to implement this test,
		// since release/3.18 is lagging behind from master in terms of netemx.
		server := filtering.NewTLSServer(filtering.TLSActionReset)
		defer server.Close()

		URL := &url.URL{
			Scheme: "https",
			Host:   server.Endpoint(),
			Path:   "/",
		}

		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		rawResp, err := sess.OONIRunFetchWithURL(sess.NewContext(), URL)
		if !strings.HasSuffix(err.Error(), "connection_reset") {
			t.Fatal("unexpected error", err)
		}
		if rawResp != "" {
			t.Fatal("expected empty raw response")
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

		URL := runtimex.Try1(url.Parse(server.URL))

		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		rawResp, err := sess.OONIRunFetchWithURL(sess.NewContext(), URL)
		if !strings.HasSuffix(err.Error(), "connection_reset") {
			t.Fatal("unexpected error", err)
		}
		if rawResp != "" {
			t.Fatal("expected empty raw response")
		}
	})
}
