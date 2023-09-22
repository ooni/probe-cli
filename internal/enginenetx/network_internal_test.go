package enginenetx

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
)

func TestNetworkUnit(t *testing.T) {
	t.Run("HTTPTransport returns the correct transport", func(t *testing.T) {
		expected := &mocks.HTTPTransport{}
		netx := &Network{txp: expected}
		if netx.HTTPTransport() != expected {
			t.Fatal("not the transport we expected")
		}
	})

	t.Run("Close calls the transport's CloseIdleConnections method", func(t *testing.T) {
		var called bool
		expected := &mocks.HTTPTransport{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		netx := &Network{txp: expected}
		if err := netx.Close(); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("did not call the transport's CloseIdleConnections")
		}
	})
}
