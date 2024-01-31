package netemx

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHTTPBinHandler(t *testing.T) {
	t.Run("missing client address", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Scheme: "http://", Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "httpbin.com",
		}
		rr := httptest.NewRecorder()
		handler := HTTPBinHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusInternalServerError {
			t.Fatal("unexpected status code", result.StatusCode)
		}
	})

	t.Run("/broken-redirect-http with client address", func(t *testing.T) {
		req := &http.Request{
			URL:        &url.URL{Scheme: "http://", Path: "/broken-redirect-http"},
			Body:       http.NoBody,
			Close:      false,
			Host:       "httpbin.com",
			RemoteAddr: net.JoinHostPort(DefaultClientAddress, "54321"),
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

	t.Run("/broken-redirect-http with another address", func(t *testing.T) {
		req := &http.Request{
			URL:        &url.URL{Scheme: "http://", Path: "/broken-redirect-http"},
			Body:       http.NoBody,
			Close:      false,
			Host:       "httpbin.com",
			RemoteAddr: net.JoinHostPort("8.8.8.8", "54321"),
		}
		rr := httptest.NewRecorder()
		handler := HTTPBinHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusFound {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "http://www.example.com/" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("/broken-redirect-https with client address", func(t *testing.T) {
		req := &http.Request{
			URL:        &url.URL{Scheme: "http://", Path: "/broken-redirect-https"},
			Body:       http.NoBody,
			Close:      false,
			Host:       "httpbin.com",
			RemoteAddr: net.JoinHostPort(DefaultClientAddress, "54321"),
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

	t.Run("/broken-redirect-https with another address", func(t *testing.T) {
		req := &http.Request{
			URL:        &url.URL{Scheme: "http://", Path: "/broken-redirect-https"},
			Body:       http.NoBody,
			Close:      false,
			Host:       "httpbin.com",
			RemoteAddr: net.JoinHostPort("8.8.8.8", "54321"),
		}
		rr := httptest.NewRecorder()
		handler := HTTPBinHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusFound {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "https://www.example.com/" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("/nonexistent URL", func(t *testing.T) {
		req := &http.Request{
			URL:        &url.URL{Scheme: "https://", Path: "/nonexistent"},
			Body:       http.NoBody,
			Close:      false,
			Host:       "httpbin.com",
			RemoteAddr: net.JoinHostPort("8.8.8.8", "54321"),
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
