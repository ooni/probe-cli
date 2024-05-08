package enginenetx

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
			Fallback: &nullPolicy{},
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
			Fallback: &nullPolicy{},
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

	t.Run("we fallback if the fallback is implemented", func(t *testing.T) {
		// define what tactic we expect to see in output
		expectTactic := &httpsDialerTactic{
			Address:        "130.192.91.211",
			InitialDelay:   0,
			Port:           "443",
			SNI:            "shelob.polito.it",
			VerifyHostname: "api.ooni.io",
		}

		// create a DNS policy where the DNS lookup fails and then add a fallback
		// use policy where we return back the expected tactic
		policy := &dnsPolicy{
			Logger: model.DiscardLogger,
			Resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, netxlite.ErrOODNSNoSuchHost
				},
			},
			Fallback: &userPolicy{
				Fallback: &nullPolicy{},
				Root: &userPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{
						"api.ooni.io:443": {expectTactic},
					},
					Version: userPolicyVersion,
				},
			},
		}

		// lookup for api.ooni.io:443
		input := policy.LookupTactics(context.Background(), "api.ooni.io", "443")

		// collect all the returned tactics
		var tactics []*httpsDialerTactic
		for tx := range input {
			tactics = append(tactics, tx)
		}

		// make sure we exactly got the tactic we expected
		if diff := cmp.Diff([]*httpsDialerTactic{expectTactic}, tactics); diff != "" {
			t.Fatal(diff)
		}
	})
}
