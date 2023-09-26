package enginenetx

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// This test ensures that a [*Network] created with [NewNetwork] collects stats.
func TestNetworkCollectsStats(t *testing.T) {
	// testcase is a test case run by this function
	type testcase struct {
		// name is the test case name
		name string

		// URL is the URL to GET
		URL string

		// initialPolicy is the initial policy to configure into the key-value store
		initialPolicy func() []byte

		// configureDPI is the function to configure DPI
		configureDPI func(dpi *netem.DPIEngine)

		// expectErr is the expected error string
		expectErr string

		// statsDomainEpnt is the domain endpoint to lookup inside the stats
		statsDomainEpnt string

		// statsTacticsSummary is the summary to lookup inside the stats
		// once we have used the statsDomain to get a record
		statsTacticsSummary string

		// expectStats contains the expected record containing tactics stats
		expectStats *statsTactic
	}

	cases := []testcase{

		{
			name: "with TCP connect failure",
			URL:  "https://api.ooni.io/",
			initialPolicy: func() []byte {
				p0 := &staticPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{
						// This policy has a different SNI and VerifyHostname, which gives
						// us confidence that the stats are using the latter
						"api.ooni.io:443": {{
							Address:        netemx.AddressApiOONIIo,
							InitialDelay:   0,
							Port:           "443",
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: staticPolicyVersion,
				}
				return runtimex.Try1(json.Marshal(p0))
			},
			configureDPI: func(dpi *netem.DPIEngine) {
				dpi.AddRule(&netem.DPICloseConnectionForServerEndpoint{
					Logger:          log.Log,
					ServerIPAddress: netemx.AddressApiOONIIo,
					ServerPort:      443,
				})
			},
			expectErr:           `Get "https://api.ooni.io/": connection_refused`,
			statsDomainEpnt:     "api.ooni.io:443",
			statsTacticsSummary: "162.55.247.208:443 sni=www.example.com verify=api.ooni.io",
			expectStats: &statsTactic{
				CountStarted:              1,
				CountTCPConnectError:      1,
				CountTLSHandshakeError:    0,
				CountTLSVerificationError: 0,
				CountSuccess:              0,
				HistoTCPConnectError: map[string]int64{
					"connection_refused": 1,
				},
				HistoTLSHandshakeError:    map[string]int64{},
				HistoTLSVerificationError: map[string]int64{},
				LastUpdated:               time.Time{},
				Tactic: &httpsDialerTactic{
					Address:        "162.55.247.208",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				},
			},
		},

		{
			name: "with TLS handshake failure",
			URL:  "https://api.ooni.io/",
			initialPolicy: func() []byte {
				p0 := &staticPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{
						// This policy has a different SNI and VerifyHostname, which gives
						// us confidence that the stats are using the latter
						"api.ooni.io:443": {{
							Address:        netemx.AddressApiOONIIo,
							InitialDelay:   0,
							Port:           "443",
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: staticPolicyVersion,
				}
				return runtimex.Try1(json.Marshal(p0))
			},
			configureDPI: func(dpi *netem.DPIEngine) {
				dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
					Logger: log.Log,
					SNI:    "www.example.com",
				})
			},
			expectErr:           `Get "https://api.ooni.io/": connection_reset`,
			statsDomainEpnt:     "api.ooni.io:443",
			statsTacticsSummary: "162.55.247.208:443 sni=www.example.com verify=api.ooni.io",
			expectStats: &statsTactic{
				CountStarted:              1,
				CountTCPConnectError:      0,
				CountTLSHandshakeError:    1,
				CountTLSVerificationError: 0,
				CountSuccess:              0,
				HistoTCPConnectError:      map[string]int64{},
				HistoTLSHandshakeError: map[string]int64{
					"connection_reset": 1,
				},
				HistoTLSVerificationError: map[string]int64{},
				LastUpdated:               time.Time{},
				Tactic: &httpsDialerTactic{
					Address:        "162.55.247.208",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				},
			},
		},

		{
			name: "with TLS verification failure",
			URL:  "https://api.ooni.io/",
			initialPolicy: func() []byte {
				p0 := &staticPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{
						// This policy has a different SNI and VerifyHostname, which gives
						// us confidence that the stats are using the latter
						"api.ooni.io:443": {{
							Address:        netemx.AddressBadSSLCom,
							InitialDelay:   0,
							Port:           "443",
							SNI:            "untrusted-root.badssl.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: staticPolicyVersion,
				}
				return runtimex.Try1(json.Marshal(p0))
			},
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr:           `Get "https://api.ooni.io/": ssl_invalid_hostname`,
			statsDomainEpnt:     "api.ooni.io:443",
			statsTacticsSummary: "104.154.89.105:443 sni=untrusted-root.badssl.com verify=api.ooni.io",
			expectStats: &statsTactic{
				CountStarted:              1,
				CountTCPConnectError:      0,
				CountTLSHandshakeError:    0,
				CountTLSVerificationError: 1,
				CountSuccess:              0,
				HistoTCPConnectError:      map[string]int64{},
				HistoTLSHandshakeError:    map[string]int64{},
				HistoTLSVerificationError: map[string]int64{
					"ssl_invalid_hostname": 1,
				},
				LastUpdated: time.Time{},
				Tactic: &httpsDialerTactic{
					Address:        "104.154.89.105",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "untrusted-root.badssl.com",
					VerifyHostname: "api.ooni.io",
				},
			},
		}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qa := netemx.MustNewScenario(netemx.InternetScenario)
			defer qa.Close()

			// make sure we apply specific DPI rules
			tc.configureDPI(qa.DPIEngine())

			// create a memory key-value store where the engine will write stats that later we
			// would be able to read to confirm we're collecting stats
			kvStore := &kvstore.Memory{}

			initialPolicy := tc.initialPolicy()
			t.Logf("initialPolicy: %s", string(initialPolicy))
			if err := kvStore.Set(staticPolicyKey, initialPolicy); err != nil {
				t.Fatal(err)
			}

			qa.Do(func() {
				byteCounter := bytecounter.New()
				resolver := netxlite.NewStdlibResolver(log.Log)

				netx := NewNetwork(byteCounter, kvStore, log.Log, nil, resolver)
				defer netx.Close()

				client := netx.NewHTTPClient()

				resp, err := client.Get(tc.URL)

				switch {
				case err == nil && tc.expectErr == "":
					// all good

				case err != nil && tc.expectErr == "":
					t.Fatal("expected", tc.expectErr, "but got", err.Error())

				case err == nil && tc.expectErr != "":
					t.Fatal("expected", tc.expectErr, "but got", err)

				case err != nil && tc.expectErr != "":
					if tc.expectErr != err.Error() {
						t.Fatal("expected", tc.expectErr, "but got", err.Error())
					}
				}

				if resp != nil {
					defer resp.Body.Close()
				}
			})

			// obtain the tactics container for the proper domain
			rawStats, err := kvStore.Get(statsKey)
			if err != nil {
				t.Fatal(err)
			}
			var rootStats statsContainer
			if err := json.Unmarshal(rawStats, &rootStats); err != nil {
				t.Fatal(err)
			}
			tactics, good := rootStats.DomainEndpoints[tc.statsDomainEpnt]
			if !good {
				t.Fatalf("no such record for `%s`", tc.statsDomainEpnt)
			}
			t.Logf("%+v", tactics)

			// we expect to see a single record
			if len(tactics.Tactics) != 1 {
				t.Fatal("expected a single tactic")
			}
			tactic, good := tactics.Tactics[tc.statsTacticsSummary]
			if !good {
				t.Fatalf("no such record for: %s", tc.statsTacticsSummary)
			}

			diffOptions := []cmp.Option{
				cmpopts.IgnoreFields(statsTactic{}, "LastUpdated"),
			}
			if diff := cmp.Diff(tc.expectStats, tactic, diffOptions...); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestLoadStatsContainer(t *testing.T) {
	type testcase struct {
		// name is the test case name
		name string

		// input returns the bytes we should Set into the key-value store
		input func() []byte

		// expectedErr is the expected error string or an empty string
		expectErr string

		// expectRoot is the expected root container content
		expectRoot *statsContainer
	}

	fourtyFiveMinutesAgo := time.Now().Add(-45 * time.Minute)

	twoWeeksAgo := time.Now().Add(-14 * 24 * time.Hour)

	cases := []testcase{{
		name: "when the key-value store does not contain any data",
		input: func() []byte {
			// Note that returning nil causes the code to NOT set anything into the kvstore
			return nil
		},
		expectErr:  "no such key",
		expectRoot: nil,
	}, {
		name: "when we cannot parse the serialized JSON",
		input: func() []byte {
			return []byte(`{`)
		},
		expectErr:  "unexpected end of JSON input",
		expectRoot: nil,
	}, {
		name: "with invalid version",
		input: func() []byte {
			return []byte(`{"Version":1}`)
		},
		expectErr:  "httpsdialerstats.state: wrong stats container version: expected=5 got=1",
		expectRoot: nil,
	}, {
		name: "on success including correct entries pruning",
		input: func() []byte {
			root := &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{
					"api.ooni.io:443": {
						Tactics: map[string]*statsTactic{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted:              4,
								CountTCPConnectError:      1,
								CountTLSHandshakeError:    1,
								CountTLSVerificationError: 1,
								CountSuccess:              1,
								HistoTCPConnectError: map[string]int64{
									"connection_refused": 1,
								},
								HistoTLSHandshakeError: map[string]int64{
									"generic_timeout_error": 1,
								},
								HistoTLSVerificationError: map[string]int64{
									"ssl_invalid_hostname": 1,
								},
								LastUpdated: fourtyFiveMinutesAgo,
								Tactic: &httpsDialerTactic{
									Address:        "162.55.247.208",
									InitialDelay:   0,
									Port:           "443",
									SNI:            "www.example.com",
									VerifyHostname: "api.ooni.io",
								},
							},
							"162.55.247.208:443 sni=www.example.org verify=api.ooni.io": { // should be skipped b/c it's old
								CountStarted:              4,
								CountTCPConnectError:      1,
								CountTLSHandshakeError:    1,
								CountTLSVerificationError: 1,
								CountSuccess:              1,
								HistoTCPConnectError: map[string]int64{
									"connection_refused": 1,
								},
								HistoTLSHandshakeError: map[string]int64{
									"generic_timeout_error": 1,
								},
								HistoTLSVerificationError: map[string]int64{
									"ssl_invalid_hostname": 1,
								},
								LastUpdated: twoWeeksAgo,
								Tactic: &httpsDialerTactic{
									Address:        "162.55.247.208",
									InitialDelay:   0,
									Port:           "443",
									SNI:            "www.example.org",
									VerifyHostname: "api.ooni.io",
								},
							},
							"162.55.247.208:443 sni=www.example.xyz verify=api.ooni.io": nil, // should be skipped because nil
							"162.55.247.208:443 sni=www.example.it verify=api.ooni.io": { // should be skipped because nil tactic
								CountStarted:              4,
								CountTCPConnectError:      1,
								CountTLSHandshakeError:    1,
								CountTLSVerificationError: 1,
								CountSuccess:              1,
								HistoTCPConnectError: map[string]int64{
									"connection_refused": 1,
								},
								HistoTLSHandshakeError: map[string]int64{
									"generic_timeout_error": 1,
								},
								HistoTLSVerificationError: map[string]int64{
									"ssl_invalid_hostname": 1,
								},
								LastUpdated: fourtyFiveMinutesAgo,
								Tactic:      nil,
							},
						},
					},
					"www.kernel.org:443": { // this whole entry should be skipped because it's too old
						Tactics: map[string]*statsTactic{
							"162.55.247.208:443 sni=www.example.com verify=www.kernel.org": {
								CountStarted:              4,
								CountTCPConnectError:      1,
								CountTLSHandshakeError:    1,
								CountTLSVerificationError: 1,
								CountSuccess:              1,
								HistoTCPConnectError: map[string]int64{
									"connection_refused": 1,
								},
								HistoTLSHandshakeError: map[string]int64{
									"generic_timeout_error": 1,
								},
								HistoTLSVerificationError: map[string]int64{
									"ssl_invalid_hostname": 1,
								},
								LastUpdated: twoWeeksAgo,
								Tactic: &httpsDialerTactic{
									Address:        "162.55.247.208",
									InitialDelay:   0,
									Port:           "443",
									SNI:            "www.example.com",
									VerifyHostname: "www.kernel.org",
								},
							},
						},
					},
					"www.kerneltrap.org:443": nil, // this whole entry should be skipped because it's nil
				},
				Version: statsContainerVersion,
			}
			return runtimex.Try1(json.Marshal(root))
		},
		expectErr: "",
		expectRoot: &statsContainer{
			DomainEndpoints: map[string]*statsDomainEndpoint{
				"api.ooni.io:443": {
					Tactics: map[string]*statsTactic{
						"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
							CountStarted:              4,
							CountTCPConnectError:      1,
							CountTLSHandshakeError:    1,
							CountTLSVerificationError: 1,
							CountSuccess:              1,
							HistoTCPConnectError: map[string]int64{
								"connection_refused": 1,
							},
							HistoTLSHandshakeError: map[string]int64{
								"generic_timeout_error": 1,
							},
							HistoTLSVerificationError: map[string]int64{
								"ssl_invalid_hostname": 1,
							},
							LastUpdated: fourtyFiveMinutesAgo,
							Tactic: &httpsDialerTactic{
								Address:        "162.55.247.208",
								InitialDelay:   0,
								Port:           "443",
								SNI:            "www.example.com",
								VerifyHostname: "api.ooni.io",
							},
						},
					},
				},
			},
			Version: statsContainerVersion,
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kvStore := &kvstore.Memory{}
			if input := tc.input(); len(input) > 0 {
				if err := kvStore.Set(statsKey, input); err != nil {
					t.Fatal(err)
				}
			}

			root, err := loadStatsContainer(kvStore)

			switch {
			case err == nil && tc.expectErr == "":
				// all good

			case err != nil && tc.expectErr == "":
				t.Fatal("expected", tc.expectErr, "but got", err.Error())

			case err == nil && tc.expectErr != "":
				t.Fatal("expected", tc.expectErr, "but got", err)

			case err != nil && tc.expectErr != "":
				if tc.expectErr != err.Error() {
					t.Fatal("expected", tc.expectErr, "but got", err.Error())
				}
			}

			if diff := cmp.Diff(tc.expectRoot, root); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestStatsManagerCallbacks(t *testing.T) {
	type testcase struct {
		name        string
		initialRoot *statsContainer
		do          func(stats *statsManager)
		expectWarnf int
		expectRoot  *statsContainer
	}

	fourtyFiveMinutesAgo := time.Now().Add(-45 * time.Minute)

	cases := []testcase{

		// When TCP connect fails and the reason is a canceled context
		{
			name: "OnTCPConnectError with ctx.Error() != nil",
			initialRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{
					"api.ooni.io:443": {
						Tactics: map[string]*statsTactic{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted: 1,
								LastUpdated:  fourtyFiveMinutesAgo,
								Tactic:       &httpsDialerTactic{}, // only required for cloning
							},
						},
					},
				},
				Version: statsContainerVersion,
			},
			do: func(stats *statsManager) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // immediately!

				tactic := &httpsDialerTactic{
					Address:        "162.55.247.208",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTCPConnectError(ctx, tactic, err)
			},
			expectWarnf: 0,
			expectRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{
					"api.ooni.io:443": {
						Tactics: map[string]*statsTactic{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted:             1,
								CountTCPConnectInterrupt: 1,
								Tactic:                   &httpsDialerTactic{},
							},
						},
					},
				},
				Version: statsContainerVersion,
			},
		},

		// When TCP connect fails and we don't already have a policy record
		{
			name: "OnTCPConnectError when we are missing the stats record for the domain",
			initialRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{},
				Version:         statsContainerVersion,
			},
			do: func(stats *statsManager) {
				ctx := context.Background()

				tactic := &httpsDialerTactic{
					Address:        "162.55.247.208",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTCPConnectError(ctx, tactic, err)
			},
			expectWarnf: 1,
			expectRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{},
				Version:         statsContainerVersion,
			},
		},

		// When TLS handshake fails and the reason is a canceled context
		{
			name: "OnTLSHandshakeError with ctx.Error() != nil",
			initialRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{
					"api.ooni.io:443": {
						Tactics: map[string]*statsTactic{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted: 1,
								LastUpdated:  fourtyFiveMinutesAgo,
								Tactic:       &httpsDialerTactic{}, // only for cloning
							},
						},
					},
				},
				Version: statsContainerVersion,
			},
			do: func(stats *statsManager) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // immediately!

				tactic := &httpsDialerTactic{
					Address:        "162.55.247.208",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTLSHandshakeError(ctx, tactic, err)
			},
			expectWarnf: 0,
			expectRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{
					"api.ooni.io:443": {
						Tactics: map[string]*statsTactic{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted:               1,
								CountTLSHandshakeInterrupt: 1,
								Tactic:                     &httpsDialerTactic{},
							},
						},
					},
				},
				Version: statsContainerVersion,
			},
		},

		// When TLS handshake fails and we don't already have a policy record
		{
			name: "OnTLSHandshakeError when we are missing the stats record for the domain",
			initialRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{},
				Version:         statsContainerVersion,
			},
			do: func(stats *statsManager) {
				ctx := context.Background()

				tactic := &httpsDialerTactic{
					Address:        "162.55.247.208",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTLSHandshakeError(ctx, tactic, err)
			},
			expectWarnf: 1,
			expectRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{},
				Version:         statsContainerVersion,
			},
		},

		// When TLS verification fails and we don't already have a policy record
		{
			name: "OnTLSVerifyError when we are missing the stats record for the domain",
			initialRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{},
				Version:         statsContainerVersion,
			},
			do: func(stats *statsManager) {
				tactic := &httpsDialerTactic{
					Address:        "162.55.247.208",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTLSVerifyError(tactic, err)
			},
			expectWarnf: 1,
			expectRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{},
				Version:         statsContainerVersion,
			},
		},

		// With success when we don't already have a policy record
		{
			name: "OnSuccess when we are missing the stats record for the domain",
			initialRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{},
				Version:         statsContainerVersion,
			},
			do: func(stats *statsManager) {
				tactic := &httpsDialerTactic{
					Address:        "162.55.247.208",
					InitialDelay:   0,
					Port:           "443",
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}

				stats.OnSuccess(tactic)
			},
			expectWarnf: 1,
			expectRoot: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{},
				Version:         statsContainerVersion,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// configure the initial value of the stats
			kvStore := &kvstore.Memory{}
			if err := kvStore.Set(statsKey, runtimex.Try1(json.Marshal(tc.initialRoot))); err != nil {
				t.Fatal(err)
			}

			// create logger counting the number Warnf invocations
			var warnfCount int
			logger := &mocks.Logger{
				MockWarnf: func(format string, v ...any) {
					warnfCount++
				},
			}

			// create the stats manager
			stats := newStatsManager(kvStore, logger)

			// invoke the proper stats callback
			tc.do(stats)

			// close the stats to trigger a kvstore write
			if err := stats.Close(); err != nil {
				t.Fatal(err)
			}

			// extract the possibly modified stats from the kvstore
			var root *statsContainer
			rawRoot, err := kvStore.Get(statsKey)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(rawRoot, &root); err != nil {
				t.Fatal(err)
			}

			// make sure the stats are the ones we expect
			diffOptions := []cmp.Option{
				cmpopts.IgnoreFields(statsTactic{}, "LastUpdated"),
				cmpopts.EquateEmpty(),
			}
			if diff := cmp.Diff(tc.expectRoot, root, diffOptions...); diff != "" {
				t.Fatal(diff)
			}

			// make sure we logged if necessary
			if tc.expectWarnf != warnfCount {
				t.Fatal("expected", tc.expectWarnf, "got", warnfCount)
			}
		})
	}
}

// Make sure that we can safely obtain statistics for a domain and a port.
func TestStatsManagerLookupTactics(t *testing.T) {

	// prepare the content of the stats
	twentyMinutesAgo := time.Now().Add(-20 * time.Minute)

	expectTactics := []*statsTactic{{
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
			Address:        "162.55.247.208",
			InitialDelay:   0,
			Port:           "443",
			SNI:            "www.repubblica.it",
			VerifyHostname: "api.ooni.io",
		},
	}, {
		CountStarted:               1,
		CountTCPConnectError:       0,
		CountTCPConnectInterrupt:   0,
		CountTLSHandshakeError:     0,
		CountTLSHandshakeInterrupt: 0,
		CountTLSVerificationError:  0,
		CountSuccess:               1,
		HistoTCPConnectError:       map[string]int64{},
		HistoTLSHandshakeError:     map[string]int64{},
		HistoTLSVerificationError:  map[string]int64{},
		LastUpdated:                twentyMinutesAgo,
		Tactic: &httpsDialerTactic{
			Address:        "162.55.247.208",
			InitialDelay:   0,
			Port:           "443",
			SNI:            "www.kernel.org",
			VerifyHostname: "api.ooni.io",
		},
	}, {
		CountStarted:               3,
		CountTCPConnectError:       0,
		CountTCPConnectInterrupt:   0,
		CountTLSHandshakeError:     0,
		CountTLSHandshakeInterrupt: 0,
		CountTLSVerificationError:  0,
		CountSuccess:               3,
		HistoTCPConnectError:       map[string]int64{},
		HistoTLSHandshakeError:     map[string]int64{},
		HistoTLSVerificationError:  map[string]int64{},
		LastUpdated:                twentyMinutesAgo,
		Tactic: &httpsDialerTactic{
			Address:        "162.55.247.208",
			InitialDelay:   0,
			Port:           "443",
			SNI:            "theconversation.com",
			VerifyHostname: "api.ooni.io",
		},
	}}

	expectContainer := &statsContainer{
		DomainEndpoints: map[string]*statsDomainEndpoint{
			"api.ooni.io:443": {
				Tactics: map[string]*statsTactic{},
			},
		},
		Version: statsContainerVersion,
	}

	for _, tactic := range expectTactics {
		expectContainer.DomainEndpoints["api.ooni.io:443"].Tactics[tactic.Tactic.tacticSummaryKey()] = tactic
	}

	// configure the initial value of the stats
	kvStore := &kvstore.Memory{}
	if err := kvStore.Set(statsKey, runtimex.Try1(json.Marshal(expectContainer))); err != nil {
		t.Fatal(err)
	}

	// create the stats manager
	stats := newStatsManager(kvStore, log.Log)

	t.Run("when we're searching for a domain endpoint we know about", func(t *testing.T) {
		// obtain tactics
		tactics, good := stats.LookupTactics("api.ooni.io", "443")
		if !good {
			t.Fatal("expected good")
		}
		if len(tactics) != 3 {
			t.Fatal("unexpected tactics length")
		}

		// sort obtained tactics lexicographically
		sort.SliceStable(tactics, func(i, j int) bool {
			return tactics[i].Tactic.tacticSummaryKey() < tactics[j].Tactic.tacticSummaryKey()
		})

		// sort the initial tactics as well
		sort.SliceStable(expectTactics, func(i, j int) bool {
			return expectTactics[i].Tactic.tacticSummaryKey() < expectTactics[j].Tactic.tacticSummaryKey()
		})

		// compare once we have sorted
		if diff := cmp.Diff(expectTactics, tactics); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("when we don't have information about a domain endpoint", func(t *testing.T) {
		// obtain tactics
		tactics, good := stats.LookupTactics("api.ooni.io", "444") // note: different port!
		if good {
			t.Fatal("expected !good")
		}
		if len(tactics) != 0 {
			t.Fatal("unexpected tactics length")
		}
	})

	t.Run("when the stats manager is manually configured to have an empty container", func(t *testing.T) {
		stats := &statsManager{
			container: &statsContainer{ /* explicitly empty */ },
			kvStore:   kvStore,
			logger:    model.DiscardLogger,
			mu:        sync.Mutex{},
		}
		tactics, good := stats.LookupTactics("api.ooni.io", "443")
		if good {
			t.Fatal("expected !good")
		}
		if len(tactics) != 0 {
			t.Fatal("unexpected tactics length")
		}
	})

	t.Run("when the stats manager is manually configured to have nil tactics", func(t *testing.T) {
		stats := &statsManager{
			container: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{
					"api.ooni.io:443": nil,
				},
				Version: 0,
			},
			kvStore: kvStore,
			logger:  model.DiscardLogger,
			mu:      sync.Mutex{},
		}
		tactics, good := stats.LookupTactics("api.ooni.io", "443")
		if good {
			t.Fatal("expected !good")
		}
		if len(tactics) != 0 {
			t.Fatal("unexpected tactics length")
		}
	})

	t.Run("when the stats manager is manually configured to have empty tactics", func(t *testing.T) {
		stats := &statsManager{
			container: &statsContainer{
				DomainEndpoints: map[string]*statsDomainEndpoint{
					"api.ooni.io:443": { /* explicitly left empty */ },
				},
				Version: 0,
			},
			kvStore: kvStore,
			logger:  model.DiscardLogger,
			mu:      sync.Mutex{},
		}
		tactics, good := stats.LookupTactics("api.ooni.io", "443")
		if good {
			t.Fatal("expected !good")
		}
		if len(tactics) != 0 {
			t.Fatal("unexpected tactics length")
		}
	})
}
