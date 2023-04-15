package resolverlookup

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestClient(t *testing.T) {
	t.Run("LookupResolverIPv4", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			c := &Client{
				Logger: model.DiscardLogger,
			}
			addr, err := c.LookupResolverIPv4(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if addr == "" {
				t.Fatal("expected a non-empty string")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			c := &Client{
				Logger: model.DiscardLogger,
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // stop immediately
			addr, err := c.LookupResolverIPv4(ctx)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("not the error we expected: %+v", err)
			}
			if len(addr) != 0 {
				t.Fatal("expected an empty address")
			}
		})
	})
}
