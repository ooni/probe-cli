package enginenetx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestNetworkUnit(t *testing.T) {
	t.Run("HTTPTransport returns the correct transport", func(t *testing.T) {
		expected := &mocks.HTTPTransport{}
		netx := &Network{txp: expected}
		if netx.HTTPTransport() != expected {
			t.Fatal("not the transport we expected")
		}
	})

	t.Run("Close calls the transport's CloseIdleConnections method", func(t *testing.T) {
		var called bool
		expected := &mocks.HTTPTransport{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		netx := &Network{
			reso: &mocks.Resolver{
				MockCloseIdleConnections: func() {
					// nothing
				},
			},
			stats: &statsManager{
				cancel:    func() { /* nothing */ },
				closeOnce: sync.Once{},
				container: &statsContainer{},
				kvStore:   &kvstore.Memory{},
				logger:    model.DiscardLogger,
				mu:        sync.Mutex{},
				pruned:    make(chan any),
				wg:        &sync.WaitGroup{},
			},
			txp: expected,
		}
		if err := netx.Close(); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("did not call the transport's CloseIdleConnections")
		}
	})

	t.Run("Close calls the resolvers's CloseIdleConnections method", func(t *testing.T) {
		var called bool
		expected := &mocks.Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		netx := &Network{
			reso: expected,
			stats: &statsManager{
				cancel:    func() { /* nothing */ },
				closeOnce: sync.Once{},
				container: &statsContainer{},
				kvStore:   &kvstore.Memory{},
				logger:    model.DiscardLogger,
				mu:        sync.Mutex{},
				pruned:    make(chan any),
				wg:        &sync.WaitGroup{},
			},
			txp: &mocks.HTTPTransport{
				MockCloseIdleConnections: func() {
					// nothing
				},
			},
		}
		if err := netx.Close(); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("did not call the resolver's CloseIdleConnections")
		}
	})

	t.Run("Close calls the .cancel field of the statsManager as a side effect", func(t *testing.T) {
		var called bool
		netx := &Network{
			reso: &mocks.Resolver{
				MockCloseIdleConnections: func() {
					// nothing
				},
			},
			stats: &statsManager{
				cancel: func() {
					called = true
				},
				closeOnce: sync.Once{},
				container: &statsContainer{},
				kvStore:   &kvstore.Memory{},
				logger:    model.DiscardLogger,
				mu:        sync.Mutex{},
				pruned:    make(chan any),
				wg:        &sync.WaitGroup{},
			},
			txp: &mocks.HTTPTransport{
				MockCloseIdleConnections: func() {
					// nothing
				},
			},
		}
		if err := netx.Close(); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("did not call the .cancel field of the statsManager")
		}
	})

	t.Run("NewNetwork uses the correct httpsDialerPolicy", func(t *testing.T) {
		// testcase is a test case run by this func
		type testcase struct {
			name         string
			kvStore      func() model.KeyValueStore
			expectStatus int
			expectBody   []byte
		}

		cases := []testcase{
			// Without a policy accessing www.example.com should lead to 200 as status
			// code and the expected web page when we're using netem
			{
				name: "when there is no user-provided policy",
				kvStore: func() model.KeyValueStore {
					return &kvstore.Memory{}
				},
				expectStatus: 200,
				expectBody:   []byte(netemx.ExampleWebPage),
			},

			// But we can create a policy that can land us on a different website (not the
			// typical use case of the policy, but definitely demonstrating it works)
			{
				name: "when there's a user-provided policy",
				kvStore: func() model.KeyValueStore {
					policy := &userPolicyRoot{
						DomainEndpoints: map[string][]*httpsDialerTactic{
							"www.example.com:443": {{
								Address:        netemx.AddressApiOONIIo,
								InitialDelay:   0,
								Port:           "443",
								SNI:            "www.example.com",
								VerifyHostname: "api.ooni.io",
							}},
						},
						Version: userPolicyVersion,
					}
					rawPolicy := runtimex.Try1(json.Marshal(policy))
					kvStore := &kvstore.Memory{}
					runtimex.Try0(kvStore.Set(userPolicyKey, rawPolicy))
					return kvStore
				},
				expectStatus: 404,
				expectBody:   []byte{},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				env := netemx.MustNewScenario(netemx.InternetScenario)
				defer env.Close()

				env.Do(func() {
					netx := NewNetwork(
						bytecounter.New(),
						tc.kvStore(),
						log.Log,
						nil, // proxy URL
						(&netxlite.Netx{}).NewStdlibResolver(log.Log),
					)
					defer netx.Close()

					client := netx.NewHTTPClient()
					resp, err := client.Get("https://www.example.com/")
					if err != nil {
						t.Fatal(err)
					}
					defer resp.Body.Close()
					if resp.StatusCode != tc.expectStatus {
						t.Fatal("StatusCode: expected", tc.expectStatus, "got", resp.StatusCode)
					}
					data, err := netxlite.ReadAllContext(context.Background(), resp.Body)
					if err != nil {
						t.Fatal(err)
					}
					if diff := cmp.Diff(tc.expectBody, data); diff != "" {
						t.Fatal(diff)
					}
				})
			})
		}
	})
}

// Make sure we get the correct policy type depending on how we call newHTTPSDialerPolicy
func TestNewHTTPSDialerPolicyTypes(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the name of the test case
		name string

		// kvStore constructs the kvstore to use
		kvStore func() model.KeyValueStore

		// proxyURL is the OPTIONAL proxy URL to use
		proxyURL *url.URL

		// expectType is the string representation of the
		// type constructed using these params
		expectType string

		// extraChecks is an OPTIONAL function that
		// will perform extra checks on the policy type
		extraChecks func(t *testing.T, root httpsDialerPolicy)
	}

	minimalUserPolicy := []byte(`{"Version":3}`)

	// this function ensures that the part dealing with stats or bridges is correct
	verifyStatsOrBridgesChain := func(t *testing.T, root *mixPolicyInterleave) {
		if root.Factor != 3 {
			t.Fatal("expected .Factory to be 3")
		}
		_ = root.Primary.(*statsPolicyV2)
		_ = root.Fallback.(*bridgesPolicyV2)

	}

	// this function ensures that the DNS ext part of the chain is correct
	verifyDNSExtChain := func(_ *testing.T, root *testHelpersPolicy) {
		_ = root.Child.(*dnsPolicy)
	}

	// this function ensures that the policy used when there's no use policy has
	// the correct type and anything below it also has the correct type
	verifyNoUserPolicyChain := func(t *testing.T, root httpsDialerPolicy) {
		interleavePolicy := root.(*mixPolicyInterleave)
		if interleavePolicy.Factor != 3 {
			t.Fatal("expected .Factory to be 3")
		}
		verifyDNSExtChain(t, interleavePolicy.Primary.(*testHelpersPolicy))
		verifyStatsOrBridgesChain(t, interleavePolicy.Fallback.(*mixPolicyInterleave))

	}

	// this function ansures that the policy used when there's an user policy has
	// the correct type and anything below it also has the correct type
	verifyUserPolicyChain := func(t *testing.T, root httpsDialerPolicy) {
		eitherOrPolicy := root.(*mixPolicyEitherOr)
		_ = eitherOrPolicy.Primary.(*userPolicyV2)
		verifyNoUserPolicyChain(t, eitherOrPolicy.Fallback)
	}

	cases := []testcase{
		{
			name: "when there is a proxy URL and there is a user policy",
			kvStore: func() model.KeyValueStore {
				store := &kvstore.Memory{}
				// this policy is mostly empty but it's enough to load
				runtimex.Try0(store.Set(userPolicyKey, minimalUserPolicy))
				return store
			},
			proxyURL: &url.URL{
				Scheme: "socks5",
				Host:   "127.0.0.1:9050",
				Path:   "/",
			},
			expectType: "*enginenetx.dnsPolicy",
		},

		{
			name: "when there is a proxy URL and there is no user policy",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: &url.URL{
				Scheme: "socks5",
				Host:   "127.0.0.1:9050",
				Path:   "/",
			},
			expectType: "*enginenetx.dnsPolicy",
		},

		{
			name: "when there is no proxy URL and there is a user policy",
			kvStore: func() model.KeyValueStore {
				store := &kvstore.Memory{}
				// this policy is mostly empty but it's enough to load
				runtimex.Try0(store.Set(userPolicyKey, minimalUserPolicy))
				return store
			},
			proxyURL:    nil,
			expectType:  "*enginenetx.mixPolicyEitherOr",
			extraChecks: verifyUserPolicyChain,
		},

		{
			name: "when there is no proxy URL and there is no user policy",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL:    nil,
			expectType:  "*enginenetx.mixPolicyInterleave",
			extraChecks: verifyNoUserPolicyChain,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			p := newHTTPSDialerPolicy(
				tc.kvStore(),
				model.DiscardLogger,
				tc.proxyURL,       // possibly nil
				&mocks.Resolver{}, // we are not using `out` so it does not matter
				&statsManager{},   // ditto
			)

			got := fmt.Sprintf("%T", p)
			if diff := cmp.Diff(tc.expectType, got); diff != "" {
				t.Fatal(diff)
			}

			if tc.extraChecks != nil {
				tc.extraChecks(t, p)
			}
		})
	}
}

// This test ensures that newHTTPSDialerPolicy is functionally working as intended.
func TestNewHTTPSDialerPolicyFunctional(t *testing.T) {
	// testcase is a test case implemented by this func
	type testcase struct {
		// name is the test case name
		name string

		// kvStore is the key-value store possibly containing user policies
		// and previous statistics about TLS endpoints
		kvStore func() model.KeyValueStore

		// proxyURL is the OPTIONAL proxy URL.
		proxyURL *url.URL

		// resolver is the DNS resolver.
		resolver model.Resolver

		// domain is the domain for which to use LookupTactics.
		domain string

		// totalExpectedEntries is the total number of entries we
		// expect the code to generate as part of this run
		totalExpectedEntries int

		// initialExpectedEntries contains the initial entries that
		// we expect to see when getting results
		initialExpectedEntries []*httpsDialerTactic
	}

	cases := []testcase{

		// Let's start with test cases in which there is no proxy and
		// no state, where we want to see that we're using the DNS, and
		// that, on top of this, we're getting bridges tactics when
		// we're using api.ooni.io and we're getting various SNIs when
		// instead we're using test helper domains.

		{
			name: "without proxy, with empty key-value store, and NXDOMAIN for www.example.com",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: nil,
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, netxlite.ErrOODNSNoSuchHost
				},
			},
			domain:                 "www.example.com",
			totalExpectedEntries:   0,
			initialExpectedEntries: nil,
		},

		{
			name: "without proxy, with empty key-value store, and addresses for www.example.com",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: nil,
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"93.184.215.14", "2606:2800:21f:cb07:6820:80da:af6b:8b2c"}, nil
				},
			},
			domain:               "www.example.com",
			totalExpectedEntries: 2,
			initialExpectedEntries: []*httpsDialerTactic{{
				Address:        "93.184.215.14",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "www.example.com",
				VerifyHostname: "www.example.com",
			}, {
				Address:        "2606:2800:21f:cb07:6820:80da:af6b:8b2c",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "www.example.com",
				VerifyHostname: "www.example.com",
			}},
		},

		{
			name: "without proxy, with empty key-value store, and NXDOMAIN for api.ooni.io",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: nil,
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, netxlite.ErrOODNSNoSuchHost
				},
			},
			domain:                 "api.ooni.io",
			totalExpectedEntries:   152,
			initialExpectedEntries: nil,
		},

		{
			name: "without proxy, with empty key-value store, and addresses for api.ooni.io",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: nil,
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"130.192.91.211", "130.192.91.231"}, nil
				},
			},
			domain:               "api.ooni.io",
			totalExpectedEntries: 154,
			initialExpectedEntries: []*httpsDialerTactic{{
				Address:        "130.192.91.211",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "api.ooni.io",
				VerifyHostname: "api.ooni.io",
			}, {
				Address:        "130.192.91.231",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "api.ooni.io",
				VerifyHostname: "api.ooni.io",
			}},
		},

		{
			name: "without proxy, with empty key-value store, and NXDOMAIN for 0.th.ooni.org",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: nil,
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, netxlite.ErrOODNSNoSuchHost
				},
			},
			domain:                 "0.th.ooni.org",
			totalExpectedEntries:   0,
			initialExpectedEntries: nil,
		},

		{
			name: "without proxy, with empty key-value store, and addresses for 0.th.ooni.org",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: nil,
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"130.192.91.211", "130.192.91.231"}, nil
				},
			},
			domain:               "0.th.ooni.org",
			totalExpectedEntries: 306,
			initialExpectedEntries: []*httpsDialerTactic{{
				Address:        "130.192.91.211",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "0.th.ooni.org",
				VerifyHostname: "0.th.ooni.org",
			}, {
				Address:        "130.192.91.231",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "0.th.ooni.org",
				VerifyHostname: "0.th.ooni.org",
			}},
		},

		// Now we repeat the same test cases but with a proxy and we want
		// to always and only see the results obtained via DNS.

		{
			name: "with proxy, with empty key-value store, and NXDOMAIN for www.example.com",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: &url.URL{}, // does not need to be filled
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, netxlite.ErrOODNSNoSuchHost
				},
			},
			domain:                 "www.example.com",
			totalExpectedEntries:   0,
			initialExpectedEntries: nil,
		},

		{
			name: "with proxy, with empty key-value store, and addresses for www.example.com",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: &url.URL{}, // does not need to be filled
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"93.184.215.14", "2606:2800:21f:cb07:6820:80da:af6b:8b2c"}, nil
				},
			},
			domain:               "www.example.com",
			totalExpectedEntries: 2,
			initialExpectedEntries: []*httpsDialerTactic{{
				Address:        "93.184.215.14",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "www.example.com",
				VerifyHostname: "www.example.com",
			}, {
				Address:        "2606:2800:21f:cb07:6820:80da:af6b:8b2c",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "www.example.com",
				VerifyHostname: "www.example.com",
			}},
		},

		{
			name: "with proxy, with empty key-value store, and NXDOMAIN for api.ooni.io",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: &url.URL{}, // does not need to be filled
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, netxlite.ErrOODNSNoSuchHost
				},
			},
			domain:                 "api.ooni.io",
			totalExpectedEntries:   0,
			initialExpectedEntries: nil,
		},

		{
			name: "without proxy, with empty key-value store, and addresses for api.ooni.io",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: &url.URL{}, // does not need to be filled
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"130.192.91.211", "130.192.91.231"}, nil
				},
			},
			domain:               "api.ooni.io",
			totalExpectedEntries: 2,
			initialExpectedEntries: []*httpsDialerTactic{{
				Address:        "130.192.91.211",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "api.ooni.io",
				VerifyHostname: "api.ooni.io",
			}, {
				Address:        "130.192.91.231",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "api.ooni.io",
				VerifyHostname: "api.ooni.io",
			}},
		},

		{
			name: "with proxy, with empty key-value store, and NXDOMAIN for 0.th.ooni.org",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: &url.URL{}, // does not need to be filled
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, netxlite.ErrOODNSNoSuchHost
				},
			},
			domain:                 "0.th.ooni.org",
			totalExpectedEntries:   0,
			initialExpectedEntries: nil,
		},

		{
			name: "with proxy, with empty key-value store, and addresses for 0.th.ooni.org",
			kvStore: func() model.KeyValueStore {
				return &kvstore.Memory{}
			},
			proxyURL: &url.URL{}, // does not need to be filled
			resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"130.192.91.211", "130.192.91.231"}, nil
				},
			},
			domain:               "0.th.ooni.org",
			totalExpectedEntries: 2,
			initialExpectedEntries: []*httpsDialerTactic{{
				Address:        "130.192.91.211",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "0.th.ooni.org",
				VerifyHostname: "0.th.ooni.org",
			}, {
				Address:        "130.192.91.231",
				InitialDelay:   0,
				Port:           "443",
				SNI:            "0.th.ooni.org",
				VerifyHostname: "0.th.ooni.org",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create manager for keeping track of statistics. This implies creating a background
			// goroutine that we'll need to close when we're done.
			stats := newStatsManager(tc.kvStore(), model.DiscardLogger, 24*time.Hour)
			defer stats.Close()

			// Create a new HTTPS dialer policy.
			policy := newHTTPSDialerPolicy(
				tc.kvStore(),
				model.DiscardLogger,
				tc.proxyURL, // possibly nil
				tc.resolver,
				stats,
			)

			// Start to generate tactics for the given domain and port.
			generator := policy.LookupTactics(context.Background(), tc.domain, "443")

			// Collect tactics
			var tactics []*httpsDialerTactic
			for entry := range generator {
				tactics = append(tactics, entry)
			}

			// To help debugging, log how many tactics we've got
			t.Log("got", len(tactics), "tactics")

			// make sure the number of expected entries is the actual number
			if len(tactics) != tc.totalExpectedEntries {
				t.Fatal("expected", tc.totalExpectedEntries, ", got", len(tactics))
			}

			// make sure we have at least N initial entries
			if len(tactics) < len(tc.initialExpectedEntries) {
				t.Fatal("expected at least", len(tc.initialExpectedEntries), "tactics, got", len(tactics))
			}

			// if we have expected initial entries, make sure they match
			if len(tc.initialExpectedEntries) > 0 {
				if diff := cmp.Diff(tc.initialExpectedEntries, tactics[:len(tc.initialExpectedEntries)]); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
