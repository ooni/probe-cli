package netemx

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestLargeFileHandler(t *testing.T) {
	t.Run("we get 500 if reading fails", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "largefile.com",
		}
		rr := httptest.NewRecorder()
		handler := LargeFileHandler(func(b []byte) (n int, err error) {
			return 0, errors.New("cannot read large file")
		})
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusInternalServerError {
			t.Fatal("unexpected status code", result.StatusCode)
		}
	})

	t.Run("otherwise we get a large file", func(t *testing.T) {
		req := &http.Request{
			URL:   &url.URL{Path: "/"},
			Body:  http.NoBody,
			Close: false,
			Host:  "yandex.com",
		}
		rr := httptest.NewRecorder()
		handler := LargeFileHandler(rand.Read)
		handler.ServeHTTP(rr, req)
		result := rr.Result()
		if result.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", result.StatusCode)
		}
		data, err := netxlite.ReadAllContext(context.Background(), result.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != 1<<25 {
			t.Fatal("cannot read the whole response body")
		}
	})
}
