package enginenetx

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestBeaconsPolicy(t *testing.T) {
	t.Run("for domains for which we don't have beacons and DNS failure", func(t *testing.T) {
		expected := errors.New("mocked error")
		policy := &BeaconsPolicy{
			Fallback: &HTTPSDialerNullPolicy{
				Logger: model.DiscardLogger,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, expected
					},
				},
			},
		}

		ctx := context.Background()
		tactics := policy.LookupTactics(ctx, "www.example.com", "443")

		var count int
		for range tactics {
			count++
		}

		if count != 0 {
			t.Fatal("expected to see zero tactics")
		}
	})

	t.Run("for domains for which we don't have beacons and DNS success", func(t *testing.T) {
		policy := &BeaconsPolicy{
			Fallback: &HTTPSDialerNullPolicy{
				Logger: model.DiscardLogger,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"93.184.216.34"}, nil
					},
				},
			},
		}

		ctx := context.Background()
		tactics := policy.LookupTactics(ctx, "www.example.com", "443")

		var count int
		for tactic := range tactics {
			count++

			host, port, err := net.SplitHostPort(tactic.Endpoint)
			if err != nil {
				t.Fatal(err)
			}
			if port != "443" {
				t.Fatal("the port should always be 443")
			}
			if host != "93.184.216.34" {
				t.Fatal("the host should always be 93.184.216.34")
			}

			if tactic.SNI != "www.example.com" {
				t.Fatal("the SNI field should always be like `www.example.com`")
			}

			if tactic.VerifyHostname != "www.example.com" {
				t.Fatal("the VerifyHostname field should always be like `www.example.com`")
			}
		}

		if count != 1 {
			t.Fatal("expected to see one tactic")
		}
	})

	t.Run("for the api.ooni.io domain", func(t *testing.T) {
		expected := errors.New("mocked error")
		policy := &BeaconsPolicy{
			Fallback: &HTTPSDialerNullPolicy{
				Logger: model.DiscardLogger,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, expected
					},
				},
			},
		}

		ctx := context.Background()
		tactics := policy.LookupTactics(ctx, "api.ooni.io", "443")

		var count int
		for tactic := range tactics {
			count++

			host, port, err := net.SplitHostPort(tactic.Endpoint)
			if err != nil {
				t.Fatal(err)
			}
			if port != "443" {
				t.Fatal("the port should always be 443")
			}
			if host != "162.55.247.208" {
				t.Fatal("the host should always be 162.55.247.208")
			}

			if tactic.SNI == "api.ooni.io" {
				t.Fatal("we should not see the `api.ooni.io` SNI on the wire")
			}

			if tactic.VerifyHostname != "api.ooni.io" {
				t.Fatal("the VerifyHostname field should always be like `api.ooni.io`")
			}
		}

		if count <= 0 {
			t.Fatal("expected to see at least one tactic")
		}
	})
}
