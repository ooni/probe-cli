package netemx

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestCloudflareHandler(t *testing.T) {
	t.Run("we get 500 with unknown client address", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "lgbt.foundation",
		}
		rr := httptest.NewRecorder()
		handler := CloudflareCAPTCHAHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusInternalServerError {
			t.Fatal("unexpected status code", result.StatusCode)
		}
	})

	t.Run("we get 503 with the default client address", func(t *testing.T) {
		req := &http.Request{
			URL:        &url.URL{Path: "/"},
			Body:       http.NoBody,
			Close:      false,
			Host:       "lgbt.foundation",
			RemoteAddr: net.JoinHostPort(DefaultClientAddress, "54321"),
		}
		rr := httptest.NewRecorder()
		handler := CloudflareCAPTCHAHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusServiceUnavailable {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		body, err := netxlite.ReadAllContext(context.Background(), result.Body)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(cloudflareCAPTCHAWebPage, body); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("we get 200 with another IP address", func(t *testing.T) {
		req := &http.Request{
			URL:        &url.URL{Path: "/"},
			Body:       http.NoBody,
			Close:      false,
			Host:       "lgbt.foundation",
			RemoteAddr: "8.8.8.8:54321",
		}
		rr := httptest.NewRecorder()
		handler := CloudflareCAPTCHAHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		body, err := netxlite.ReadAllContext(context.Background(), result.Body)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]byte(ExampleWebPage), body); diff != "" {
			t.Fatal(diff)
		}
	})
}
