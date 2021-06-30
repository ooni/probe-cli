package netxmocks

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

func TestHTTPTransportRoundTrip(t *testing.T) {
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
}

func TestHTTPTransportCloseIdleConnections(t *testing.T) {
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
}
