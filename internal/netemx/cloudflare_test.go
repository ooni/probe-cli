package netemx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestCloudflareHandler(t *testing.T) {
	t.Run("we get the expected webpage", func(t *testing.T) {
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
}
