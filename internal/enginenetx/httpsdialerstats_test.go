package enginenetx_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/enginenetx"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestHTTPSDialerCollectStats(t *testing.T) {
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

		// statsDomain is the domain to lookup inside the stats
		statsDomain string

		// statsTacticsSummary is the summary to lookup inside the stats
		// once we have used the statsDomain to get a record
		statsTacticsSummary string

		// expectStats contains the expected record containing tactics stats
		expectStats *enginenetx.HTTPSDialerStatsTacticRecord
	}

	cases := []testcase{

		{
			name: "with TCP connect failure",
			URL:  "https://api.ooni.io/",
			initialPolicy: func() []byte {
				p0 := &enginenetx.HTTPSDialerStaticPolicyRoot{
					Domains: map[string][]*enginenetx.HTTPSDialerTactic{
						// This policy has a different SNI and VerifyHostname, which gives
						// us confidence that the stats are using the latter
						"api.ooni.io": {{
							Endpoint:       net.JoinHostPort(netemx.AddressApiOONIIo, "443"),
							InitialDelay:   0,
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: enginenetx.HTTPSDialerStaticPolicyVersion,
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
			statsDomain:         "api.ooni.io",
			statsTacticsSummary: "162.55.247.208:443 sni=www.example.com verify=api.ooni.io",
			expectStats: &enginenetx.HTTPSDialerStatsTacticRecord{
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
				Tactic: &enginenetx.HTTPSDialerTactic{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				},
			},
		},

		{
			name: "with TLS handshake failure",
			URL:  "https://api.ooni.io/",
			initialPolicy: func() []byte {
				p0 := &enginenetx.HTTPSDialerStaticPolicyRoot{
					Domains: map[string][]*enginenetx.HTTPSDialerTactic{
						// This policy has a different SNI and VerifyHostname, which gives
						// us confidence that the stats are using the latter
						"api.ooni.io": {{
							Endpoint:       net.JoinHostPort(netemx.AddressApiOONIIo, "443"),
							InitialDelay:   0,
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: enginenetx.HTTPSDialerStaticPolicyVersion,
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
			statsDomain:         "api.ooni.io",
			statsTacticsSummary: "162.55.247.208:443 sni=www.example.com verify=api.ooni.io",
			expectStats: &enginenetx.HTTPSDialerStatsTacticRecord{
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
				Tactic: &enginenetx.HTTPSDialerTactic{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				},
			},
		},

		{
			name: "with TLS verification failure",
			URL:  "https://api.ooni.io/",
			initialPolicy: func() []byte {
				p0 := &enginenetx.HTTPSDialerStaticPolicyRoot{
					Domains: map[string][]*enginenetx.HTTPSDialerTactic{
						// This policy has a different SNI and VerifyHostname, which gives
						// us confidence that the stats are using the latter
						"api.ooni.io": {{
							Endpoint:       net.JoinHostPort(netemx.AddressBadSSLCom, "443"),
							InitialDelay:   0,
							SNI:            "untrusted-root.badssl.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: enginenetx.HTTPSDialerStaticPolicyVersion,
				}
				return runtimex.Try1(json.Marshal(p0))
			},
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr:           `Get "https://api.ooni.io/": ssl_invalid_hostname`,
			statsDomain:         "api.ooni.io",
			statsTacticsSummary: "104.154.89.105:443 sni=untrusted-root.badssl.com verify=api.ooni.io",
			expectStats: &enginenetx.HTTPSDialerStatsTacticRecord{
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
				Tactic: &enginenetx.HTTPSDialerTactic{
					Endpoint:       "104.154.89.105:443",
					InitialDelay:   0,
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
			if err := kvStore.Set(enginenetx.HTTPSDialerStaticPolicyKey, initialPolicy); err != nil {
				t.Fatal(err)
			}

			qa.Do(func() {
				byteCounter := bytecounter.New()
				resolver := netxlite.NewStdlibResolver(log.Log)

				netx := enginenetx.NewNetwork(byteCounter, kvStore, log.Log, nil, resolver)
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
			rawStats, err := kvStore.Get(enginenetx.HTTPSDialerStatsKey)
			if err != nil {
				t.Fatal(err)
			}
			var rootStats enginenetx.HTTPSDialerStatsRootContainer
			if err := json.Unmarshal(rawStats, &rootStats); err != nil {
				t.Fatal(err)
			}
			tactics, good := rootStats.Domains[tc.statsDomain]
			if !good {
				t.Fatalf("no such record for `%s`", tc.statsDomain)
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
				cmpopts.IgnoreFields(enginenetx.HTTPSDialerStatsTacticRecord{}, "LastUpdated"),
			}
			if diff := cmp.Diff(tc.expectStats, tactic, diffOptions...); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
