package enginenetx

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestStatsPolicyWorkingAsIntended(t *testing.T) {
	// prepare the content of the stats
	twentyMinutesAgo := time.Now().Add(-20 * time.Minute)

	const beaconAddress = netemx.AddressApiOONIIo

	expectTacticsStats := []*statsTactic{{
		CountStarted:               5,
		CountTCPConnectError:       0,
		CountTCPConnectInterrupt:   0,
		CountTLSHandshakeError:     0,
		CountTLSHandshakeInterrupt: 0,
		CountTLSVerificationError:  0,
		CountSuccess:               5, // this one always succeeds, so it should be there
		HistoTCPConnectError:       map[string]int64{},
		HistoTLSHandshakeError:     map[string]int64{},
		HistoTLSVerificationError:  map[string]int64{},
		LastUpdated:                twentyMinutesAgo,
		Tactic: &httpsDialerTactic{
			Address:        beaconAddress,
			InitialDelay:   0,
			Port:           "443",
			SNI:            "www.repubblica.it",
			VerifyHostname: "api.ooni.io",
		},
	}, {
		CountStarted:               3,
		CountTCPConnectError:       0,
		CountTCPConnectInterrupt:   0,
		CountTLSHandshakeError:     1,
		CountTLSHandshakeInterrupt: 0,
		CountTLSVerificationError:  0,
		CountSuccess:               2, // this one sometimes succeded so it should be added
		HistoTCPConnectError:       map[string]int64{},
		HistoTLSHandshakeError:     map[string]int64{},
		HistoTLSVerificationError:  map[string]int64{},
		LastUpdated:                twentyMinutesAgo,
		Tactic: &httpsDialerTactic{
			Address:        beaconAddress,
			InitialDelay:   0,
			Port:           "443",
			SNI:            "www.kernel.org",
			VerifyHostname: "api.ooni.io",
		},
	}, {
		CountStarted:               3,
		CountTCPConnectError:       0,
		CountTCPConnectInterrupt:   0,
		CountTLSHandshakeError:     3, // this one always failed, so should not be added
		CountTLSHandshakeInterrupt: 0,
		CountTLSVerificationError:  0,
		CountSuccess:               0,
		HistoTCPConnectError:       map[string]int64{},
		HistoTLSHandshakeError:     map[string]int64{},
		HistoTLSVerificationError:  map[string]int64{},
		LastUpdated:                twentyMinutesAgo,
		Tactic: &httpsDialerTactic{
			Address:        beaconAddress,
			InitialDelay:   0,
			Port:           "443",
			SNI:            "theconversation.com",
			VerifyHostname: "api.ooni.io",
		},
	}, {
		CountStarted:               4,
		CountTCPConnectError:       0,
		CountTCPConnectInterrupt:   0,
		CountTLSHandshakeError:     0,
		CountTLSHandshakeInterrupt: 0,
		CountTLSVerificationError:  0,
		CountSuccess:               4,
		HistoTCPConnectError:       map[string]int64{},
		HistoTLSHandshakeError:     map[string]int64{},
		HistoTLSVerificationError:  map[string]int64{},
		LastUpdated:                twentyMinutesAgo,
		Tactic:                     nil, // the nil policy here should cause this entry to be filtered out
	}, {
		CountStarted:               0,
		CountTCPConnectError:       0,
		CountTCPConnectInterrupt:   0,
		CountTLSHandshakeError:     0,
		CountTLSHandshakeInterrupt: 0,
		CountTLSVerificationError:  0,
		CountSuccess:               0,
		HistoTCPConnectError:       map[string]int64{},
		HistoTLSHandshakeError:     map[string]int64{},
		HistoTLSVerificationError:  map[string]int64{},
		LastUpdated:                time.Time{}, // the zero time should exclude this one
		Tactic: &httpsDialerTactic{
			Address:        beaconAddress,
			InitialDelay:   0,
			Port:           "443",
			SNI:            "ilpost.it",
			VerifyHostname: "api.ooni.io",
		},
	}}

	// createStatsManager creates a stats manager given some baseline stats
	createStatsManager := func(domainEndpoint string, tactics ...*statsTactic) *statsManager {
		container := &statsContainer{
			DomainEndpoints: map[string]*statsDomainEndpoint{
				domainEndpoint: {
					Tactics: map[string]*statsTactic{},
				},
			},
			Version: statsContainerVersion,
		}

		for _, tx := range tactics {
			if tx.Tactic != nil {
				container.DomainEndpoints[domainEndpoint].Tactics[tx.Tactic.tacticSummaryKey()] = tx
			}
		}

		kvStore := &kvstore.Memory{}
		if err := kvStore.Set(statsKey, runtimex.Try1(json.Marshal(container))); err != nil {
			t.Fatal(err)
		}

		const trimInterval = 30 * time.Second
		return newStatsManager(kvStore, log.Log, trimInterval)
	}

	t.Run("when we have unique statistics", func(t *testing.T) {
		// create stats manager
		stats := createStatsManager("api.ooni.io:443", expectTacticsStats...)
		defer stats.Close()

		// create the composed policy
		policy := &statsPolicy{
			Fallback: &dnsPolicy{
				Logger: log.Log,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						switch domain {
						case "api.ooni.io":
							return []string{beaconAddress}, nil
						default:
							return nil, netxlite.ErrOODNSNoSuchHost
						}
					},
				},
			},
			Stats: stats,
		}

		// obtain the tactics from the saved stats
		var tactics []*httpsDialerTactic
		for entry := range policy.LookupTactics(context.Background(), "api.ooni.io", "443") {
			tactics = append(tactics, entry)
		}

		// compute the list of results we expect to see from the stats data
		var expect []*httpsDialerTactic
		idx := 0
		for _, entry := range expectTacticsStats {
			if entry.CountSuccess <= 0 || entry.Tactic == nil {
				continue // we SHOULD NOT include entries that systematically failed
			}
			t := entry.Tactic.Clone()
			t.InitialDelay = happyEyeballsDelay(idx)
			expect = append(expect, t)
			idx++
		}

		// extend the expected list to include DNS results
		expect = append(expect, &httpsDialerTactic{
			Address:        beaconAddress,
			InitialDelay:   2 * time.Second,
			Port:           "443",
			SNI:            "api.ooni.io",
			VerifyHostname: "api.ooni.io",
		})

		// perform the actual comparison
		if diff := cmp.Diff(expect, tactics); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("when we have duplicates", func(t *testing.T) {
		// add each entry twice to create obvious duplicates
		statsWithDupes := []*statsTactic{}
		for _, entry := range expectTacticsStats {
			statsWithDupes = append(statsWithDupes, entry.Clone())
			statsWithDupes = append(statsWithDupes, entry.Clone())
		}

		// create stats manager
		stats := createStatsManager("api.ooni.io:443", statsWithDupes...)
		defer stats.Close()

		// create the composed policy
		policy := &statsPolicy{
			Fallback: &dnsPolicy{
				Logger: log.Log,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						switch domain {
						case "api.ooni.io":
							// Twice so we try to cause duplicate entries also with the DNS policy
							return []string{beaconAddress, beaconAddress}, nil
						default:
							return nil, netxlite.ErrOODNSNoSuchHost
						}
					},
				},
			},
			Stats: stats,
		}

		// obtain the tactics from the saved stats
		var tactics []*httpsDialerTactic
		for entry := range policy.LookupTactics(context.Background(), "api.ooni.io", "443") {
			tactics = append(tactics, entry)
		}

		// compute the list of results we expect to see from the stats data
		var expect []*httpsDialerTactic
		idx := 0
		for _, entry := range expectTacticsStats {
			if entry.CountSuccess <= 0 || entry.Tactic == nil {
				continue // we SHOULD NOT include entries that systematically failed
			}
			t := entry.Tactic.Clone()
			t.InitialDelay = happyEyeballsDelay(idx)
			expect = append(expect, t)
			idx++
		}

		// extend the expected list to include DNS results
		expect = append(expect, &httpsDialerTactic{
			Address:        beaconAddress,
			InitialDelay:   2 * time.Second,
			Port:           "443",
			SNI:            "api.ooni.io",
			VerifyHostname: "api.ooni.io",
		})

		// perform the actual comparison
		if diff := cmp.Diff(expect, tactics); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("we avoid manipulating nil tactics", func(t *testing.T) {
		// create stats manager
		stats := createStatsManager("api.ooni.io:443", expectTacticsStats...)
		defer stats.Close()

		// create the composed policy
		policy := &statsPolicy{
			Fallback: &mocksPolicy{
				MockLookupTactics: func(ctx context.Context, domain, port string) <-chan *httpsDialerTactic {
					out := make(chan *httpsDialerTactic)
					go func() {
						defer close(out)

						// explicitly send nil on the channel
						out <- nil
					}()
					return out
				},
			},
			Stats: stats,
		}

		// obtain the tactics from the saved stats
		var tactics []*httpsDialerTactic
		for entry := range policy.LookupTactics(context.Background(), "api.ooni.io", "443") {
			tactics = append(tactics, entry)
		}

		// compute the list of results we expect to see from the stats data
		var expect []*httpsDialerTactic
		idx := 0
		for _, entry := range expectTacticsStats {
			if entry.CountSuccess <= 0 || entry.Tactic == nil {
				continue // we SHOULD NOT include entries that systematically failed
			}
			t := entry.Tactic.Clone()
			t.InitialDelay = happyEyeballsDelay(idx)
			expect = append(expect, t)
			idx++
		}

		// perform the actual comparison
		if diff := cmp.Diff(expect, tactics); diff != "" {
			t.Fatal(diff)
		}
	})
}

type mocksPolicy struct {
	MockLookupTactics func(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic
}

var _ httpsDialerPolicy = &mocksPolicy{}

// LookupTactics implements httpsDialerPolicy.
func (p *mocksPolicy) LookupTactics(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	return p.MockLookupTactics(ctx, domain, port)
}

func TestStatsPolicyPostProcessTactics(t *testing.T) {
	t.Run("we do nothing when good is false", func(t *testing.T) {
		tactics := statsPolicyPostProcessTactics(nil, false)
		if len(tactics) != 0 {
			t.Fatal("expected zero-lenght return value")
		}
	})

	t.Run("we filter out cases in which t or t.Tactic are nil", func(t *testing.T) {
		expected := &statsTactic{}
		ff := &testingx.FakeFiller{}
		ff.Fill(&expected)

		input := []*statsTactic{nil, {
			CountStarted:               0,
			CountTCPConnectError:       0,
			CountTCPConnectInterrupt:   0,
			CountTLSHandshakeError:     0,
			CountTLSHandshakeInterrupt: 0,
			CountTLSVerificationError:  0,
			CountSuccess:               0,
			HistoTCPConnectError:       map[string]int64{},
			HistoTLSHandshakeError:     map[string]int64{},
			HistoTLSVerificationError:  map[string]int64{},
			LastUpdated:                time.Time{},
			Tactic:                     nil,
		}, nil, expected}

		got := statsPolicyPostProcessTactics(input, true)

		if len(got) != 1 {
			t.Fatal("expected just one element")
		}

		if diff := cmp.Diff(expected.Tactic, got[0]); diff != "" {
			t.Fatal(diff)
		}
	})
}
