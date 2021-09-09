package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

func TestRoundTripper(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		expected := errors.New("mocked error")
		txp := &RoundTripper{
			MockRoundTrip: func(ctx context.Context, query []byte) ([]byte, error) {
				return nil, expected
			},
		}
		resp, err := txp.RoundTrip(context.Background(), make([]byte, 16))
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if resp != nil {
			t.Fatal("expected nil response here")
		}
	})

	t.Run("RequiresPadding", func(t *testing.T) {
		txp := &RoundTripper{
			MockRequiresPadding: func() bool {
				return true
			},
		}
		if txp.RequiresPadding() != true {
			t.Fatal("unexpected result")
		}
	})

	t.Run("Network", func(t *testing.T) {
		txp := &RoundTripper{
			MockNetwork: func() string {
				return "antani"
			},
		}
		if txp.Network() != "antani" {
			t.Fatal("unexpected result")
		}
	})

	t.Run("Address", func(t *testing.T) {
		txp := &RoundTripper{
			MockAddress: func() string {
				return "mascetti"
			},
		}
		if txp.Address() != "mascetti" {
			t.Fatal("unexpected result")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		called := &atomicx.Int64{}
		txp := &RoundTripper{
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
