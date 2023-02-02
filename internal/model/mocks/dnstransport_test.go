package mocks

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestDNSTransport(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		expected := errors.New("mocked error")
		txp := &DNSTransport{
			MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
				return nil, expected
			},
		}
		resp, err := txp.RoundTrip(context.Background(), &DNSQuery{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if resp != nil {
			t.Fatal("expected nil response here")
		}
	})

	t.Run("RequiresPadding", func(t *testing.T) {
		txp := &DNSTransport{
			MockRequiresPadding: func() bool {
				return true
			},
		}
		if txp.RequiresPadding() != true {
			t.Fatal("unexpected result")
		}
	})

	t.Run("Network", func(t *testing.T) {
		txp := &DNSTransport{
			MockNetwork: func() string {
				return "antani"
			},
		}
		if txp.Network() != "antani" {
			t.Fatal("unexpected result")
		}
	})

	t.Run("Address", func(t *testing.T) {
		txp := &DNSTransport{
			MockAddress: func() string {
				return "mascetti"
			},
		}
		if txp.Address() != "mascetti" {
			t.Fatal("unexpected result")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		called := &atomic.Int64{}
		txp := &DNSTransport{
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
