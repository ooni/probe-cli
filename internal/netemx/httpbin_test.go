package netemx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHTTPBinHandler(t *testing.T) {
	t.Run("/broken-redirect with http", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Scheme: "http://", Path: "/broken-redirect"},
			Body:  http.NoBody,
			Close: false,
			Host:  "httpbin.com",
		}
		rr := httptest.NewRecorder()
		handler := HTTPBinHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusFound {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "http://" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("/broken-redirect with https", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Scheme: "https://", Path: "/broken-redirect"},
			Body:  http.NoBody,
			Close: false,
			Host:  "httpbin.com",
		}
		rr := httptest.NewRecorder()
		handler := HTTPBinHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusFound {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "https://" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("/", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Scheme: "https://", Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "httpbin.com",
		}
		rr := httptest.NewRecorder()
		handler := HTTPBinHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusNotFound {
			t.Fatal("unexpected status code", result.StatusCode)
		}
	})
}
