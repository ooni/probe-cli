package enginenetx

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestHTTPSDialerStaticPolicy(t *testing.T) {
	t.Run("NewHTTPSDialerStaticPolicy", func(t *testing.T) {
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
			expectedPolicy *HTTPSDialerStaticPolicy
		}

		fallback := &HTTPSDialerNullPolicy{}

		cases := []testcase{{
			name:           "when there is no key in the kvstore",
			key:            "",
			input:          []byte{},
			expectErr:      "no such key",
			expectedPolicy: nil,
		}, {
			name:           "with nil input",
			key:            HTTPSDialerStaticPolicyKey,
			input:          nil,
			expectErr:      "hujson: line 1, column 1: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with invalid serialized JSON",
			key:            HTTPSDialerStaticPolicyKey,
			input:          []byte(`{`),
			expectErr:      "hujson: line 1, column 2: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with empty JSON",
			key:            HTTPSDialerStaticPolicyKey,
			input:          []byte(`{}`),
			expectErr:      "httpsdialerstatic.conf: wrong static policy version: expected=1 got=0",
			expectedPolicy: nil,
		}, {
			name: "with real serialized policy",
			key:  HTTPSDialerStaticPolicyKey,
			input: (func() []byte {
				return runtimex.Try1(json.Marshal(&HTTPSDialerStaticPolicyRoot{
					Domains: map[string][]*HTTPSDialerTactic{
						"api.ooni.io": {{
							Endpoint:       "162.55.247.208:443",
							InitialDelay:   0,
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Endpoint:       "46.101.82.151:443",
							InitialDelay:   300 * time.Millisecond,
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Endpoint:       "[2a03:b0c0:1:d0::ec4:9001]:443",
							InitialDelay:   600 * time.Millisecond,
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Endpoint:       "46.101.82.151:443",
							InitialDelay:   3000 * time.Millisecond,
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}, {
							Endpoint:       "[2a03:b0c0:1:d0::ec4:9001]:443",
							InitialDelay:   3300 * time.Millisecond,
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: HTTPSDialerStaticPolicyVersion,
				}))
			})(),
			expectErr: "",
			expectedPolicy: &HTTPSDialerStaticPolicy{
				Fallback: fallback,
				Root: &HTTPSDialerStaticPolicyRoot{
					Domains: map[string][]*HTTPSDialerTactic{
						"api.ooni.io": {{
							Endpoint:       "162.55.247.208:443",
							InitialDelay:   0,
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Endpoint:       "46.101.82.151:443",
							InitialDelay:   300 * time.Millisecond,
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Endpoint:       "[2a03:b0c0:1:d0::ec4:9001]:443",
							InitialDelay:   600 * time.Millisecond,
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, {
							Endpoint:       "46.101.82.151:443",
							InitialDelay:   3000 * time.Millisecond,
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}, {
							Endpoint:       "[2a03:b0c0:1:d0::ec4:9001]:443",
							InitialDelay:   3300 * time.Millisecond,
							SNI:            "www.example.com",
							VerifyHostname: "api.ooni.io",
						}},
					},
					Version: HTTPSDialerStaticPolicyVersion,
				},
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				kvStore := &kvstore.Memory{}
				runtimex.Try0(kvStore.Set(tc.key, tc.input))

				policy, err := NewHTTPSDialerStaticPolicy(kvStore, fallback)

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
		expectedTactic := &HTTPSDialerTactic{
			Endpoint:       "162.55.247.208:443",
			InitialDelay:   0,
			SNI:            "www.example.com",
			VerifyHostname: "api.ooni.io",
		}
		staticPolicyRoot := &HTTPSDialerStaticPolicyRoot{
			Domains: map[string][]*HTTPSDialerTactic{
				"api.ooni.io": {expectedTactic},
			},
			Version: HTTPSDialerStaticPolicyVersion,
		}
		kvStore := &kvstore.Memory{}
		rawStaticPolicyRoot := runtimex.Try1(json.Marshal(staticPolicyRoot))
		if err := kvStore.Set(HTTPSDialerStaticPolicyKey, rawStaticPolicyRoot); err != nil {
			t.Fatal(err)
		}

		t.Run("with canceled context and static policy", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately cancel

			policy, err := NewHTTPSDialerStaticPolicy(kvStore, nil /* explictly to crash if used */)
			if err != nil {
				t.Fatal(err)
			}

			tactics := policy.LookupTactics(ctx, "api.ooni.io", "443")
			got := []*HTTPSDialerTactic{}
			for tactic := range tactics {
				t.Logf("%+v", tactic)
				got = append(got, tactic)
			}

			switch value := len(got); value {
			case 0:
				// the context arm was immediately selected

			case 1:
				// the sender warm was selected first
				if diff := cmp.Diff(expectedTactic, got[0]); diff != "" {
					t.Fatal(diff)
				}

			default:
				panic(fmt.Sprintf("unexpected len(got): %d", value))
			}
		})

		t.Run("with static policy", func(t *testing.T) {
			ctx := context.Background()

			policy, err := NewHTTPSDialerStaticPolicy(kvStore, nil /* explictly to crash if used */)
			if err != nil {
				t.Fatal(err)
			}

			tactics := policy.LookupTactics(ctx, "api.ooni.io", "443")
			got := []*HTTPSDialerTactic{}
			for tactic := range tactics {
				t.Logf("%+v", tactic)
				got = append(got, tactic)
			}

			switch value := len(got); value {
			case 1:
				if diff := cmp.Diff(expectedTactic, got[0]); diff != "" {
					t.Fatal(diff)
				}

			default:
				panic(fmt.Sprintf("unexpected len(got): %d", value))
			}
		})

		t.Run("with canceled context and fallback policy", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately cancel

			fallback := &HTTPSDialerNullPolicy{
				Logger: log.Log,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"93.184.216.34"}, nil
					},
				},
			}

			policy, err := NewHTTPSDialerStaticPolicy(kvStore, fallback)
			if err != nil {
				t.Fatal(err)
			}

			tactics := policy.LookupTactics(ctx, "www.example.com", "443")
			got := []*HTTPSDialerTactic{}
			for tactic := range tactics {
				t.Logf("%+v", tactic)
				got = append(got, tactic)
			}

			switch value := len(got); value {
			case 0:
				// the context arm was immediately selected or the resolved failed

			case 1:
				// the arm returning a tactic won the race
				expect := &HTTPSDialerTactic{
					Endpoint:       "93.184.216.34:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "www.example.com",
				}
				if diff := cmp.Diff(expect, got[0]); diff != "" {
					t.Fatal(diff)
				}

			default:
				panic(fmt.Sprintf("unexpected len(got): %d", value))
			}
		})

		t.Run("with fallback policy", func(t *testing.T) {
			ctx := context.Background()

			fallback := &HTTPSDialerNullPolicy{
				Logger: log.Log,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"93.184.216.34"}, nil
					},
				},
			}

			policy, err := NewHTTPSDialerStaticPolicy(kvStore, fallback)
			if err != nil {
				t.Fatal(err)
			}

			tactics := policy.LookupTactics(ctx, "www.example.com", "443")
			got := []*HTTPSDialerTactic{}
			for tactic := range tactics {
				t.Logf("%+v", tactic)
				got = append(got, tactic)
			}

			switch value := len(got); value {
			case 1:
				expect := &HTTPSDialerTactic{
					Endpoint:       "93.184.216.34:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "www.example.com",
				}
				if diff := cmp.Diff(expect, got[0]); diff != "" {
					t.Fatal(diff)
				}

			default:
				panic(fmt.Sprintf("unexpected len(got): %d", value))
			}
		})
	})
}
