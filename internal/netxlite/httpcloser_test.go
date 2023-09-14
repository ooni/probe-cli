package netxlite

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
)

func TestHTTPTransportConnectionsCloser(t *testing.T) {
	t.Run("CloseIdleConnections", func(t *testing.T) {
		var (
			calledTxp    bool
			calledDialer bool
			calledTLS    bool
		)
		txp := &httpTransportConnectionsCloser{
			HTTPTransport: &mocks.HTTPTransport{
				MockCloseIdleConnections: func() {
					calledTxp = true
				},
			},
			Dialer: &mocks.Dialer{
				MockCloseIdleConnections: func() {
					calledDialer = true
				},
			},
			TLSDialer: &mocks.TLSDialer{
				MockCloseIdleConnections: func() {
					calledTLS = true
				},
			},
		}
		txp.CloseIdleConnections()
		if !calledDialer || !calledTLS || !calledTxp {
			t.Fatal("not called")
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		expected := errors.New("mocked error")
		txp := &httpTransportConnectionsCloser{
			HTTPTransport: &mocks.HTTPTransport{
				MockRoundTrip: func(req *http.Request) (*http.Response, error) {
					return nil, expected
				},
			},
		}
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://www.google.com")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("unexpected resp")
		}
	})
}
