package enginenetx

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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
			key:            httpsDialerStaticPolicyKey,
			input:          nil,
			expectErr:      "hujson: line 1, column 1: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with invalid serialized JSON",
			key:            httpsDialerStaticPolicyKey,
			input:          []byte(`{`),
			expectErr:      "hujson: line 1, column 2: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with empty JSON",
			key:            httpsDialerStaticPolicyKey,
			input:          []byte(`{}`),
			expectErr:      "httpsdialer.conf: wrong static policy version: expected=1 got=0",
			expectedPolicy: nil,
		}, {
			name: "with real serialized policy",
			key:  httpsDialerStaticPolicyKey,
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
					Version: httpsDialerStaticPolicyVersion,
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
					Version: httpsDialerStaticPolicyVersion,
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
		t.Run("we can lookup a static tactic", func(t *testing.T) {
			expect := []*HTTPSDialerTactic{
				{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.com",
					VerifyHostname: "api.ooni.io",
				},
				{
					Endpoint:       "162.55.247.208:443",
					InitialDelay:   0,
					SNI:            "www.example.org",
					VerifyHostname: "api.ooni.io",
				},
			}

			p := &HTTPSDialerStaticPolicy{
				Fallback: nil, // explicitly such that there is a panic if we access it
				Root: &HTTPSDialerStaticPolicyRoot{
					Domains: map[string][]*HTTPSDialerTactic{
						"api.ooni.io": expect,
					},
					Version: httpsDialerStaticPolicyVersion,
				},
			}

			ctx := context.Background()
			resolver := &mocks.Resolver{} // empty to cause panic if any method is invoked
			got, err := p.LookupTactics(ctx, "api.ooni.io", "443", resolver)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("we fallback if needed", func(t *testing.T) {
			expect := errors.New("mocked error")

			resolver := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, expect
				},
			}

			p := &HTTPSDialerStaticPolicy{
				Fallback: &HTTPSDialerNullPolicy{},
				Root: &HTTPSDialerStaticPolicyRoot{
					Domains: nil, // empty so we fallback for all domains
					Version: httpsDialerStaticPolicyVersion,
				},
			}

			ctx := context.Background()
			tactics, err := p.LookupTactics(ctx, "api.ooni.io", "443", resolver)
			if !errors.Is(err, expect) {
				t.Fatal("unexpected error", err)
			}

			if len(tactics) != 0 {
				t.Fatal("expected no tactics here")
			}
		})
	})

	t.Run("Parallelism", func(t *testing.T) {
		p := &HTTPSDialerStaticPolicy{ /* empty */ }
		if p.Parallelism() != 16 {
			t.Fatal("unexpected parallelism")
		}
	})
}
