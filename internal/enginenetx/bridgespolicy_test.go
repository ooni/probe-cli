package enginenetx

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestBridgesPolicy(t *testing.T) {
	t.Run("for domains for which we don't have bridges and DNS failure", func(t *testing.T) {
		expected := errors.New("mocked error")
		p := &bridgesPolicy{
			Fallback: &dnsPolicy{
				Logger: model.DiscardLogger,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, expected
					},
				},
			},
		}

		ctx := context.Background()
		tactics := p.LookupTactics(ctx, "www.example.com", "443")

		var count int
		for range tactics {
			count++
		}

		if count != 0 {
			t.Fatal("expected to see zero tactics")
		}
	})

	t.Run("for domains for which we don't have bridges and DNS success", func(t *testing.T) {
		p := &bridgesPolicy{
			Fallback: &dnsPolicy{
				Logger: model.DiscardLogger,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"93.184.216.34"}, nil
					},
				},
			},
		}

		ctx := context.Background()
		tactics := p.LookupTactics(ctx, "www.example.com", "443")

		var count int
		for tactic := range tactics {
			count++

			if tactic.Port != "443" {
				t.Fatal("the port should always be 443")
			}
			if tactic.Address != "93.184.216.34" {
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

	// TODO(bassosimone): we need to write better test cases for what
	// happens when we have a mixture of tactics here.

	t.Run("for the api.ooni.io domain with DNS failure", func(t *testing.T) {
		expected := errors.New("mocked error")
		p := &bridgesPolicy{
			Fallback: &dnsPolicy{
				Logger: model.DiscardLogger,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, expected
					},
				},
			},
		}

		ctx := context.Background()
		tactics := p.LookupTactics(ctx, "api.ooni.io", "443")

		var count int
		for tactic := range tactics {
			count++

			if tactic.Port != "443" {
				t.Fatal("the port should always be 443")
			}
			if tactic.Address != "162.55.247.208" {
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

	t.Run("for test helper domains", func(t *testing.T) {
		for _, domain := range bridgesPolicyTestHelpersDomains {
			t.Run(domain, func(t *testing.T) {
				expectedAddrs := []string{"164.92.180.7"}

				p := &bridgesPolicy{
					Fallback: &dnsPolicy{
						Logger: model.DiscardLogger,
						Resolver: &mocks.Resolver{
							MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
								return expectedAddrs, nil
							},
						},
					},
				}

				ctx := context.Background()
				index := 0
				for tactics := range p.LookupTactics(ctx, domain, "443") {

					if tactics.Address != "164.92.180.7" {
						t.Fatal("unexpected .Address")
					}

					if tactics.InitialDelay != happyEyeballsDelay(index) {
						t.Fatal("unexpected .InitialDelay")
					}
					index++

					if tactics.Port != "443" {
						t.Fatal("unexpected .Port")
					}

					if tactics.SNI == domain {
						t.Fatal("unexpected .Domain")
					}

					if tactics.VerifyHostname != domain {
						t.Fatal("unexpected .VerifyHostname")
					}
				}
			})
		}
	})
}
