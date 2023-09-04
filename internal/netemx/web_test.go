package netemx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestExampleWebPageHandler(t *testing.T) {
	t.Run("we're redirected if the host is example.com", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "example.com",
		}
		rr := httptest.NewRecorder()
		handler := ExampleWebPageHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusPermanentRedirect {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "https://www.example.com/" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("we're redirected if the host is example.org", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "example.org",
		}
		rr := httptest.NewRecorder()
		handler := ExampleWebPageHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusPermanentRedirect {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "https://www.example.org/" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("we get a 400 for an unknown host", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "antani.xyz",
		}
		rr := httptest.NewRecorder()
		handler := ExampleWebPageHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code", result.StatusCode)
		}
	})
}
