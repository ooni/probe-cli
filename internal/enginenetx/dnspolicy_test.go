package enginenetx

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestDNSPolicy(t *testing.T) {
	t.Run("LookupTactics with canceled context", func(t *testing.T) {
		var called int

		policy := &dnsPolicy{
			Logger: &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					called++
				},
			},
			Resolver: &mocks.Resolver{}, // empty so we crash if we hit the resolver
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel!

		tactics := policy.LookupTactics(ctx, "www.example.com", "443")

		var count int
		for range tactics {
			count++
		}

		if count != 0 {
			t.Fatal("expected to see no tactic")
		}
		if called != 1 {
			t.Fatal("did not call Debugf")
		}
	})

	t.Run("we short circuit IP addresses", func(t *testing.T) {
		policy := &dnsPolicy{
			Logger:   model.DiscardLogger,
			Resolver: &mocks.Resolver{}, // empty so we crash if we hit the resolver
		}

		tactics := policy.LookupTactics(context.Background(), "130.192.91.211", "443")

		var count int
		for tactic := range tactics {
			count++

			if tactic.Address != "130.192.91.211" {
				t.Fatal("invalid endpoint address")
			}
			if tactic.InitialDelay != 0 {
				t.Fatal("unexpected .InitialDelay")
			}
			if tactic.Port != "443" {
				t.Fatal("invalid endpoint port")
			}
			if tactic.SNI != "130.192.91.211" {
				t.Fatal("invalid SNI")
			}
			if tactic.VerifyHostname != "130.192.91.211" {
				t.Fatal("invalid VerifyHostname")
			}
		}

		if count != 1 {
			t.Fatal("expected to see just one tactic")
		}
	})
}
