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
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestStaticPolicy(t *testing.T) {
	t.Run("newStaticPolicy", func(t *testing.T) {
		// testcase is a test case implemented by this function
		type testcase struct {
			// name is the test case name
			name string

			// key is the key to use for settings the input inside the kvstore
			key string

			// input contains the serialized input bytes
			input []byte

			// expectErr contains the expected error string or the empty string on success
			expectErr string

			// expectRoot contains the expected policy we loaded or nil
			expectedPolicy *staticPolicy
		}

		fallback := &dnsPolicy{}

		cases := []testcase{{
			name:           "when there is no key in the kvstore",
			key:            "",
			input:          []byte{},
			expectErr:      "no such key",
			expectedPolicy: nil,
		}, {
			name:           "with nil input",
			key:            staticPolicyKey,
			input:          nil,
			expectErr:      "hujson: line 1, column 1: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with invalid serialized JSON",
			key:            staticPolicyKey,
			input:          []byte(`{`),
			expectErr:      "hujson: line 1, column 2: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with empty JSON",
			key:            staticPolicyKey,
			input:          []byte(`{}`),
			expectErr:      "httpsdialerstatic.conf: wrong static policy version: expected=3 got=0",
			expectedPolicy: nil,
		}, {
			name: "with real serialized policy",
			key:  staticPolicyKey,
			input: (func() []byte {
				return runtimex.Try1(json.Marshal(&staticPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{
						"api.ooni.io:443": {{
							Address:        "162.55.247.208",
							InitialDelay:   0,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Address:        "46.101.82.151",
							InitialDelay:   300 * time.Millisecond,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Address:        "2a03:b0c0:1:d0::ec4:9001",
							InitialDelay:   600 * time.Millisecond,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Address:        "46.101.82.151",
							InitialDelay:   3000 * time.Millisecond,
							Port:           "443",
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}, {
							Address:        "2a03:b0c0:1:d0::ec4:9001",
							InitialDelay:   3300 * time.Millisecond,
							Port:           "443",
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: staticPolicyVersion,
				}))
			})(),
			expectErr: "",
			expectedPolicy: &staticPolicy{
				Fallback: fallback,
				Root: &staticPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{
						"api.ooni.io:443": {{
							Address:        "162.55.247.208",
							InitialDelay:   0,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Address:        "46.101.82.151",
							InitialDelay:   300 * time.Millisecond,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Address:        "2a03:b0c0:1:d0::ec4:9001",
							InitialDelay:   600 * time.Millisecond,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Address:        "46.101.82.151",
							InitialDelay:   3000 * time.Millisecond,
							Port:           "443",
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}, {
							Address:        "2a03:b0c0:1:d0::ec4:9001",
							InitialDelay:   3300 * time.Millisecond,
							Port:           "443",
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: staticPolicyVersion,
				},
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				kvStore := &kvstore.Memory{}
				runtimex.Try0(kvStore.Set(tc.key, tc.input))

				policy, err := newStaticPolicy(kvStore, fallback)

				switch {
				case err != nil && tc.expectErr == "":
					t.Fatal("expected", tc.expectErr, "got", err)

				case err == nil && tc.expectErr != "":
					t.Fatal("expected", tc.expectErr, "got", err)

				case err != nil && tc.expectErr != "":
					if diff := cmp.Diff(tc.expectErr, err.Error()); diff != "" {
						t.Fatal(diff)
					}

				case err == nil && tc.expectErr == "":
					// all good
				}

				if diff := cmp.Diff(tc.expectedPolicy, policy); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	t.Run("LookupTactics", func(t *testing.T) {
		expectedTactic := &httpsDialerTactic{
			Address:        "162.55.247.208",
			InitialDelay:   0,
			Port:           "443",
			SNI:            "www.example.com",
			VerifyHostname: "api.ooni.io",
		}
		staticPolicyRoot := &staticPolicyRoot{
			DomainEndpoints: map[string][]*httpsDialerTactic{
				"api.ooni.io:443": {expectedTactic},
			},
			Version: staticPolicyVersion,
		}
		kvStore := &kvstore.Memory{}
		rawStaticPolicyRoot := runtimex.Try1(json.Marshal(staticPolicyRoot))
		if err := kvStore.Set(staticPolicyKey, rawStaticPolicyRoot); err != nil {
			t.Fatal(err)
		}

		t.Run("with static policy", func(t *testing.T) {
			ctx := context.Background()

			policy, err := newStaticPolicy(kvStore, nil /* explictly to crash if used */)
			if err != nil {
				t.Fatal(err)
			}

			tactics := policy.LookupTactics(ctx, "api.ooni.io", "443")
			got := []*httpsDialerTactic{}
			for tactic := range tactics {
				t.Logf("%+v", tactic)
				got = append(got, tactic)
			}

			expect := []*httpsDialerTactic{expectedTactic}

			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("we fallback if needed", func(t *testing.T) {
			ctx := context.Background()

			fallback := &dnsPolicy{
				Logger: log.Log,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"93.184.216.34"}, nil
					},
				},
			}

			policy, err := newStaticPolicy(kvStore, fallback)
			if err != nil {
				t.Fatal(err)
			}

			tactics := policy.LookupTactics(ctx, "www.example.com", "443")
			got := []*httpsDialerTactic{}
			for tactic := range tactics {
				t.Logf("%+v", tactic)
				got = append(got, tactic)
			}

			expect := []*httpsDialerTactic{{
				Address:        "93.184.216.34",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "www.example.com",
				VerifyHostname: "www.example.com",
			}}

			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}
