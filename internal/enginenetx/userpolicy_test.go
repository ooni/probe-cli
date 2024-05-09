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

func TestUserPolicyV2(t *testing.T) {
	t.Run("newUserPolicyV2", func(t *testing.T) {
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
			expectedPolicy *userPolicyV2
		}

		cases := []testcase{{
			name:           "when there is no key in the kvstore",
			key:            "",
			input:          []byte{},
			expectErr:      "no such key",
			expectedPolicy: nil,
		}, {
			name:           "with nil input",
			key:            userPolicyKey,
			input:          nil,
			expectErr:      "hujson: line 1, column 1: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with invalid serialized JSON",
			key:            userPolicyKey,
			input:          []byte(`{`),
			expectErr:      "hujson: line 1, column 2: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with empty JSON",
			key:            userPolicyKey,
			input:          []byte(`{}`),
			expectErr:      "bridges.conf: wrong user policy version: expected=3 got=0",
			expectedPolicy: nil,
		}, {
			name: "with real serialized policy",
			key:  userPolicyKey,
			input: (func() []byte {
				return runtimex.Try1(json.Marshal(&userPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{

						// Please, note how the input includes explicitly nil entries
						// with the purpose of making sure the code can handle them
						"api.ooni.io:443": {{
							Address:        "162.55.247.208",
							InitialDelay:   0,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, nil, {
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
						}, nil, {
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
						}, nil},
						//

					},
					Version: userPolicyVersion,
				}))
			})(),
			expectErr: "",
			expectedPolicy: &userPolicyV2{
				Root: &userPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{
						"api.ooni.io:443": {{
							Address:        "162.55.247.208",
							InitialDelay:   0,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, nil, {
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
						}, nil, {
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
						}, nil},
					},
					Version: userPolicyVersion,
				},
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				kvStore := &kvstore.Memory{}
				runtimex.Try0(kvStore.Set(tc.key, tc.input))

				policy, err := newUserPolicyV2(kvStore)

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
		// define the tactic we would expect to see
		expectedTactic := &httpsDialerTactic{
			Address:        "162.55.247.208",
			InitialDelay:   0,
			Port:           "443",
			SNI:            "www.example.com",
			VerifyHostname: "api.ooni.io",
		}

		// define the root of the user policy
		userPolicyRoot := &userPolicyRoot{
			DomainEndpoints: map[string][]*httpsDialerTactic{
				// Note that here we're adding explicitly nil entries
				// to make sure that the code correctly handles 'em
				"api.ooni.io:443": {
					nil,
					expectedTactic,
					nil,
				},

				// We add additional entries to make sure that in those
				// cases we are going to get nil entries as they're basically
				// empty and so non-actionable for us.
				"api.ooni.xyz:443": nil,
				"api.ooni.org:443": {},
				"api.ooni.com:443": {nil, nil, nil},
			},
			Version: userPolicyVersion,
		}

		// serialize into a key-value store running in memory
		kvStore := &kvstore.Memory{}
		rawUserPolicyRoot := runtimex.Try1(json.Marshal(userPolicyRoot))
		if err := kvStore.Set(userPolicyKey, rawUserPolicyRoot); err != nil {
			t.Fatal(err)
		}

		t.Run("with user policy", func(t *testing.T) {
			ctx := context.Background()

			policy, err := newUserPolicyV2(kvStore)
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

		t.Run("we get nothing if there is no entry in the user policy", func(t *testing.T) {
			ctx := context.Background()

			policy, err := newUserPolicyV2(kvStore)
			if err != nil {
				t.Fatal(err)
			}

			tactics := policy.LookupTactics(ctx, "www.example.com", "443")
			got := []*httpsDialerTactic{}
			for tactic := range tactics {
				t.Logf("%+v", tactic)
				got = append(got, tactic)
			}

			expect := []*httpsDialerTactic{}

			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("we get nothing if the entry in the user policy is ~empty", func(t *testing.T) {
			ctx := context.Background()

			policy, err := newUserPolicyV2(kvStore)
			if err != nil {
				t.Fatal(err)
			}

			// these cases are specially constructed to be empty/invalid user policies
			for _, domain := range []string{"api.ooni.xyz", "api.ooni.org", "api.ooni.com"} {
				t.Run(domain, func(t *testing.T) {
					tactics := policy.LookupTactics(ctx, domain, "443")
					got := []*httpsDialerTactic{}
					for tactic := range tactics {
						t.Logf("%+v", tactic)
						got = append(got, tactic)
					}

					expect := []*httpsDialerTactic{}

					if diff := cmp.Diff(expect, got); diff != "" {
						t.Fatal(diff)
					}
				})
			}
		})
	})
}

func TestUserPolicy(t *testing.T) {
	t.Run("newUserPolicy", func(t *testing.T) {
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
			expectedPolicy *userPolicy
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
			key:            userPolicyKey,
			input:          nil,
			expectErr:      "hujson: line 1, column 1: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with invalid serialized JSON",
			key:            userPolicyKey,
			input:          []byte(`{`),
			expectErr:      "hujson: line 1, column 2: parsing value: unexpected EOF",
			expectedPolicy: nil,
		}, {
			name:           "with empty JSON",
			key:            userPolicyKey,
			input:          []byte(`{}`),
			expectErr:      "bridges.conf: wrong user policy version: expected=3 got=0",
			expectedPolicy: nil,
		}, {
			name: "with real serialized policy",
			key:  userPolicyKey,
			input: (func() []byte {
				return runtimex.Try1(json.Marshal(&userPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{

						// Please, note how the input includes explicitly nil entries
						// with the purpose of making sure the code can handle them
						"api.ooni.io:443": {{
							Address:        "162.55.247.208",
							InitialDelay:   0,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, nil, {
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
						}, nil, {
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
						}, nil},
						//

					},
					Version: userPolicyVersion,
				}))
			})(),
			expectErr: "",
			expectedPolicy: &userPolicy{
				Fallback: fallback,
				Root: &userPolicyRoot{
					DomainEndpoints: map[string][]*httpsDialerTactic{
						"api.ooni.io:443": {{
							Address:        "162.55.247.208",
							InitialDelay:   0,
							Port:           "443",
							SNI:            "api.ooni.io",
							VerifyHostname: "api.ooni.io",
						}, nil, {
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
						}, nil, {
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
						}, nil},
					},
					Version: userPolicyVersion,
				},
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				kvStore := &kvstore.Memory{}
				runtimex.Try0(kvStore.Set(tc.key, tc.input))

				policy, err := newUserPolicy(kvStore, fallback)

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
		userPolicyRoot := &userPolicyRoot{
			DomainEndpoints: map[string][]*httpsDialerTactic{
				// Note that here we're adding explicitly nil entries
				// to make sure that the code correctly handles 'em
				"api.ooni.io:443": {
					nil,
					expectedTactic,
					nil,
				},

				// We add additional entries to make sure that in those
				// cases we are going to fallback as they're basically empty
				// and so non-actionable for us.
				"api.ooni.xyz:443": nil,
				"api.ooni.org:443": {},
				"api.ooni.com:443": {nil, nil, nil},
			},
			Version: userPolicyVersion,
		}
		kvStore := &kvstore.Memory{}
		rawUserPolicyRoot := runtimex.Try1(json.Marshal(userPolicyRoot))
		if err := kvStore.Set(userPolicyKey, rawUserPolicyRoot); err != nil {
			t.Fatal(err)
		}

		t.Run("with user policy", func(t *testing.T) {
			ctx := context.Background()

			policy, err := newUserPolicy(kvStore, nil /* explictly to crash if used */)
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

		t.Run("we fallback if there is no entry in the user policy", func(t *testing.T) {
			ctx := context.Background()

			fallback := &dnsPolicy{
				Logger: log.Log,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"93.184.216.34"}, nil
					},
				},
			}

			policy, err := newUserPolicy(kvStore, fallback)
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

		t.Run("we fallback if the entry in the user policy is ~empty", func(t *testing.T) {
			ctx := context.Background()

			fallback := &dnsPolicy{
				Logger: log.Log,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"93.184.216.34"}, nil
					},
				},
			}

			policy, err := newUserPolicy(kvStore, fallback)
			if err != nil {
				t.Fatal(err)
			}

			// these cases are specially constructed to be empty/invalid user policies
			for _, domain := range []string{"api.ooni.xyz", "api.ooni.org", "api.ooni.com"} {
				t.Run(domain, func(t *testing.T) {
					tactics := policy.LookupTactics(ctx, domain, "443")
					got := []*httpsDialerTactic{}
					for tactic := range tactics {
						t.Logf("%+v", tactic)
						got = append(got, tactic)
					}

					expect := []*httpsDialerTactic{{
						Address:        "93.184.216.34",
						InitialDelay:   0,
						Port:           "443",
						SNI:            domain,
						VerifyHostname: domain,
					}}

					if diff := cmp.Diff(expect, got); diff != "" {
						t.Fatal(diff)
					}
				})
			}
		})
	})
}
