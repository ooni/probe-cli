package enginenetx

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestLoadHTTPSDialerStatsRootContainer(t *testing.T) {
	type testcase struct {
		// name is the test case name
		name string

		// input returns the bytes we should Set into the key-value store
		input func() []byte

		// expectedErr is the expected error string or an empty string
		expectErr string

		// expectRoot is the expected root container content
		expectRoot *HTTPSDialerStatsRootContainer
	}

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
		expectErr:  "httpsdialerstats.state: wrong stats container version: expected=2 got=1",
		expectRoot: nil,
	}, {
		name: "on success",
		input: func() []byte {
			root := &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{
					"api.ooni.io": {
						Tactics: map[string]*HTTPSDialerStatsTacticRecord{
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
								LastUpdated: time.Date(2023, 9, 25, 0, 0, 0, 0, time.UTC),
								Tactic: &HTTPSDialerTactic{
									Endpoint:       "162.55.247.208:443",
									InitialDelay:   0,
									SNI:            "www.example.com",
									VerifyHostname: "api.ooni.io",
								},
							},
						},
					},
				},
				Version: HTTPSDialerStatsContainerVersion,
			}
			return runtimex.Try1(json.Marshal(root))
		},
		expectErr: "",
		expectRoot: &HTTPSDialerStatsRootContainer{
			Domains: map[string]*HTTPSDialerStatsTacticsContainer{
				"api.ooni.io": {
					Tactics: map[string]*HTTPSDialerStatsTacticRecord{
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
							LastUpdated: time.Date(2023, 9, 25, 0, 0, 0, 0, time.UTC),
							Tactic: &HTTPSDialerTactic{
								Endpoint:       "162.55.247.208:443",
								InitialDelay:   0,
								SNI:            "www.example.com",
								VerifyHostname: "api.ooni.io",
							},
						},
					},
				},
			},
			Version: HTTPSDialerStatsContainerVersion,
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kvStore := &kvstore.Memory{}
			if input := tc.input(); len(input) > 0 {
				if err := kvStore.Set(HTTPSDialerStatsKey, input); err != nil {
					t.Fatal(err)
				}
			}

			root, err := loadHTTPSDialerStatsRootContainer(kvStore)

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

func TestHTTPSDialerStatsManagerCallbacks(t *testing.T) {
	type testcase struct {
		name        string
		initialRoot *HTTPSDialerStatsRootContainer
		do          func(stats *HTTPSDialerStatsManager)
		expectWarnf int
		expectRoot  *HTTPSDialerStatsRootContainer
	}

	cases := []testcase{

		// When TCP connect fails and the reason is a canceled context
		{
			name: "OnTCPConnectError with ctx.Error() != nil",
			initialRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{
					"api.ooni.io": {
						Tactics: map[string]*HTTPSDialerStatsTacticRecord{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted: 1,
							},
						},
					},
				},
				Version: HTTPSDialerStatsContainerVersion,
			},
			do: func(stats *HTTPSDialerStatsManager) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // immediately!

				tactic := &HTTPSDialerTactic{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTCPConnectError(ctx, tactic, err)
			},
			expectWarnf: 0,
			expectRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{
					"api.ooni.io": {
						Tactics: map[string]*HTTPSDialerStatsTacticRecord{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted:             1,
								CountTCPConnectInterrupt: 1,
							},
						},
					},
				},
				Version: HTTPSDialerStatsContainerVersion,
			},
		},

		// When TCP connect fails and we don't already have a policy record
		{
			name: "OnTCPConnectError when we are missing the stats record for the domain",
			initialRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
				Version: HTTPSDialerStatsContainerVersion,
			},
			do: func(stats *HTTPSDialerStatsManager) {
				ctx := context.Background()

				tactic := &HTTPSDialerTactic{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTCPConnectError(ctx, tactic, err)
			},
			expectWarnf: 1,
			expectRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
				Version: HTTPSDialerStatsContainerVersion,
			},
		},

		// When TLS handshake fails and the reason is a canceled context
		{
			name: "OnTLSHandshakeError with ctx.Error() != nil",
			initialRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{
					"api.ooni.io": {
						Tactics: map[string]*HTTPSDialerStatsTacticRecord{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted: 1,
							},
						},
					},
				},
				Version: HTTPSDialerStatsContainerVersion,
			},
			do: func(stats *HTTPSDialerStatsManager) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // immediately!

				tactic := &HTTPSDialerTactic{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTLSHandshakeError(ctx, tactic, err)
			},
			expectWarnf: 0,
			expectRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{
					"api.ooni.io": {
						Tactics: map[string]*HTTPSDialerStatsTacticRecord{
							"162.55.247.208:443 sni=www.example.com verify=api.ooni.io": {
								CountStarted:               1,
								CountTLSHandshakeInterrupt: 1,
							},
						},
					},
				},
				Version: HTTPSDialerStatsContainerVersion,
			},
		},

		// When TLS handshake fails and we don't already have a policy record
		{
			name: "OnTLSHandshakeError when we are missing the stats record for the domain",
			initialRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
				Version: HTTPSDialerStatsContainerVersion,
			},
			do: func(stats *HTTPSDialerStatsManager) {
				ctx := context.Background()

				tactic := &HTTPSDialerTactic{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTLSHandshakeError(ctx, tactic, err)
			},
			expectWarnf: 1,
			expectRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
				Version: HTTPSDialerStatsContainerVersion,
			},
		},

		// When TLS verification fails and we don't already have a policy record
		{
			name: "OnTLSVerifyError when we are missing the stats record for the domain",
			initialRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
				Version: HTTPSDialerStatsContainerVersion,
			},
			do: func(stats *HTTPSDialerStatsManager) {
				tactic := &HTTPSDialerTactic{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}
				err := errors.New("generic_timeout_error")

				stats.OnTLSVerifyError(tactic, err)
			},
			expectWarnf: 1,
			expectRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
				Version: HTTPSDialerStatsContainerVersion,
			},
		},

		// With success when we don't already have a policy record
		{
			name: "OnSuccess when we are missing the stats record for the domain",
			initialRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
				Version: HTTPSDialerStatsContainerVersion,
			},
			do: func(stats *HTTPSDialerStatsManager) {
				tactic := &HTTPSDialerTactic{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				}

				stats.OnSuccess(tactic)
			},
			expectWarnf: 1,
			expectRoot: &HTTPSDialerStatsRootContainer{
				Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
				Version: HTTPSDialerStatsContainerVersion,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// configure the initial value of the stats
			kvStore := &kvstore.Memory{}
			if err := kvStore.Set(HTTPSDialerStatsKey, runtimex.Try1(json.Marshal(tc.initialRoot))); err != nil {
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
			stats := NewHTTPSDialerStatsManager(kvStore, logger)

			// invoke the proper stats callback
			tc.do(stats)

			// close the stats to trigger a kvstore write
			if err := stats.Close(); err != nil {
				t.Fatal(err)
			}

			// extract the possibly modified stats from the kvstore
			var root *HTTPSDialerStatsRootContainer
			rawRoot, err := kvStore.Get(HTTPSDialerStatsKey)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(rawRoot, &root); err != nil {
				t.Fatal(err)
			}

			// make sure the stats are the ones we expect
			diffOptions := []cmp.Option{
				cmpopts.IgnoreFields(HTTPSDialerStatsTacticRecord{}, "LastUpdated"),
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
