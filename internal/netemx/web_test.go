package netemx

import (
	"fmt"
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

func TestURLShortenerFactory(t *testing.T) {
	handler := URLShortenerFactory(DefaultURLShortenerMapping).NewHandler(nil, nil)

	for key, value := range DefaultURLShortenerMapping {
		t.Run(fmt.Sprintf("for %s => %s", key, value), func(t *testing.T) {
			rr := httptest.NewRecorder()

			req := &http.Request{
				URL: &url.URL{
					Path: key,
				},
			}

			handler.ServeHTTP(rr, req)

			res := rr.Result()

			if res.StatusCode != http.StatusPermanentRedirect {
				t.Fatal("unexpected StatusCode", res.StatusCode)
			}
			loc := res.Header.Get("Location")
			if loc != value {
				t.Fatal("expected", value, "got", loc)
			}
		})
	}

	t.Run("for nonexistent mapping", func(t *testing.T) {
		rr := httptest.NewRecorder()

		req := &http.Request{
			URL: &url.URL{
				Path: "/antani",
			},
		}

		handler.ServeHTTP(rr, req)

		res := rr.Result()

		if res.StatusCode != http.StatusNotFound {
			t.Fatal("unexpected StatusCode", res.StatusCode)
		}
		loc := res.Header.Get("Location")
		if loc != "" {
			t.Fatal("expected", `""`, "got", loc)
		}
	})
}
