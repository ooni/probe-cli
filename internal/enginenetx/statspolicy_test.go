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
		CountSuccess:               5,
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
		CountSuccess:               2,
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
		CountTLSHandshakeError:     3,
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
			container.DomainEndpoints[domainEndpoint].Tactics[tx.Tactic.tacticSummaryKey()] = tx
		}

		kvStore := &kvstore.Memory{}
		if err := kvStore.Set(statsKey, runtimex.Try1(json.Marshal(container))); err != nil {
			t.Fatal(err)
		}

		return newStatsManager(kvStore, log.Log)
	}

	t.Run("when we have unique statistics", func(t *testing.T) {
		// create stats manager
		stats := createStatsManager("api.ooni.io:443", expectTacticsStats...)

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
		for idx, entry := range expectTacticsStats {
			t := entry.Tactic.Clone()
			t.InitialDelay = happyEyeballsDelay(idx)
			expect = append(expect, t)
		}

		// extend the expected list to include DNS results
		expect = append(expect, &httpsDialerTactic{
			Address:        beaconAddress,
			InitialDelay:   4 * time.Second,
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
		for idx, entry := range expectTacticsStats {
			t := entry.Tactic.Clone()
			t.InitialDelay = happyEyeballsDelay(idx)
			expect = append(expect, t)
		}

		// extend the expected list to include DNS results
		expect = append(expect, &httpsDialerTactic{
			Address:        beaconAddress,
			InitialDelay:   4 * time.Second,
			Port:           "443",
			SNI:            "api.ooni.io",
			VerifyHostname: "api.ooni.io",
		})

		// perform the actual comparison
		if diff := cmp.Diff(expect, tactics); diff != "" {
			t.Fatal(diff)
		}
	})
}