package mocks

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestHTTPTransport(t *testing.T) {
	t.Run("Network", func(t *testing.T) {
		expected := "udp"
		txp := &HTTPTransport{
			MockNetwork: func() string {
				return expected
			},
		}
		if txp.Network() != expected {
			t.Fatal("unexpected network value")
		}
	})

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
		called := &atomic.Int64{}
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
		called := &atomic.Int64{}
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

func TestHTTPResponseWriter(t *testing.T) {
	t.Run("Header", func(t *testing.T) {
		expect := http.Header{}
		w := &HTTPResponseWriter{
			MockHeader: func() http.Header {
				return expect
			},
		}
		got := w.Header()
		got.Set("Content-Type", "text/plain")
		if expect.Get("Content-Type") != "text/plain" {
			t.Fatal("we didn't get the expected header value")
		}
	})

	t.Run("Write", func(t *testing.T) {
		expected := errors.New("mocked error")
		w := &HTTPResponseWriter{
			MockWrite: func(b []byte) (int, error) {
				return 0, expected
			},
		}
		buffer := make([]byte, 16)
		count, err := w.Write(buffer)
		if count != 0 {
			t.Fatal("invalid count")
		}
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("WriteHeader", func(t *testing.T) {
		var called bool
		w := &HTTPResponseWriter{
			MockWriteHeader: func(statusCode int) {
				called = true
			},
		}
		w.WriteHeader(200)
		if !called {
			t.Fatal("not called")
		}
	})
}
