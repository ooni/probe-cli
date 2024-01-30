package netemx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestYandexHandler(t *testing.T) {
	t.Run("we're redirected if the host is xn--d1acpjx3f.xn--p1ai", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "xn--d1acpjx3f.xn--p1ai",
		}
		rr := httptest.NewRecorder()
		handler := YandexHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusPermanentRedirect {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "https://yandex.com/" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("we're redirected if the host is yandex.com", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "yandex.com",
		}
		rr := httptest.NewRecorder()
		handler := YandexHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusPermanentRedirect {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "https://ya.ru/" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("we correctly handle the presence of a port", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "yandex.com:80",
		}
		rr := httptest.NewRecorder()
		handler := YandexHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusPermanentRedirect {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		if loc := result.Header.Get("Location"); loc != "https://ya.ru/" {
			t.Fatal("unexpected location", loc)
		}
	})

	t.Run("we get 200 for ya.ru", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "ya.ru",
		}
		rr := httptest.NewRecorder()
		handler := YandexHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", result.StatusCode)
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
		handler := YandexHandler()
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code", result.StatusCode)
		}
	})
}
