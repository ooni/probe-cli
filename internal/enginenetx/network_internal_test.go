package enginenetx

import (
	"sync"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
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
		netx := &Network{
			reso: &mocks.Resolver{
				MockCloseIdleConnections: func() {
					// nothing
				},
			},
			stats: &HTTPSDialerStatsManager{
				TimeNow: time.Now,
				kvStore: &kvstore.Memory{},
				logger:  model.DiscardLogger,
				mu:      sync.Mutex{},
				root:    &HTTPSDialerStatsRootContainer{},
			},
			txp: expected,
		}
		if err := netx.Close(); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("did not call the transport's CloseIdleConnections")
		}
	})

	t.Run("Close calls the resolvers's CloseIdleConnections method", func(t *testing.T) {
		var called bool
		expected := &mocks.Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		netx := &Network{
			reso: expected,
			stats: &HTTPSDialerStatsManager{
				TimeNow: time.Now,
				kvStore: &kvstore.Memory{},
				logger:  model.DiscardLogger,
				mu:      sync.Mutex{},
				root:    &HTTPSDialerStatsRootContainer{},
			},
			txp: &mocks.HTTPTransport{
				MockCloseIdleConnections: func() {
					// nothing
				},
			},
		}
		if err := netx.Close(); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("did not call the transport's CloseIdleConnections")
		}
	})
}
