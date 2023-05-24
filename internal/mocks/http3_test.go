package mocks

import (
	"errors"
	"net/http"
	"testing"
)

func TestHTTP3RoundTripper(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		expected := errors.New("mocked error")
		txp := &HTTP3RoundTripper{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, expected
			},
		}
		resp, err := txp.RoundTrip(&http.Request{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("unexpected resp")
		}
	})

	t.Run("Close", func(t *testing.T) {
		expected := errors.New("mocked error")
		txp := &HTTP3RoundTripper{
			MockClose: func() error {
				return expected
			},
		}
		if err := txp.Close(); !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})
}
