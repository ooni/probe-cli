package mocks

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

func TestHTTPTransport(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		expected := errors.New("mocked error")
		txp := &HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, expected
			},
		}
		resp, err := txp.RoundTrip(&http.Request{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if resp != nil {
			t.Fatal("expected nil response here")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		called := &atomicx.Int64{}
		txp := &HTTPTransport{
			MockCloseIdleConnections: func() {
				called.Add(1)
			},
		}
		txp.CloseIdleConnections()
		if called.Load() != 1 {
			t.Fatal("not called")
		}
	})
}

func TestHTTPClient(t *testing.T) {
	t.Run("Do", func(t *testing.T) {
		expected := errors.New("mocked error")
		clnt := &HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				return nil, expected
			},
		}
		resp, err := clnt.Do(&http.Request{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if resp != nil {
			t.Fatal("expected nil response here")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		called := &atomicx.Int64{}
		clnt := &HTTPClient{
			MockCloseIdleConnections: func() {
				called.Add(1)
			},
		}
		clnt.CloseIdleConnections()
		if called.Load() != 1 {
			t.Fatal("not called")
		}
	})
}
