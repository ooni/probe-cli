package enginenetx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"testing"

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
				container: &statsContainer{},
				kvStore:   &kvstore.Memory{},
				logger:    model.DiscardLogger,
				mu:        sync.Mutex{},
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
				container: &statsContainer{},
				kvStore:   &kvstore.Memory{},
				logger:    model.DiscardLogger,
				mu:        sync.Mutex{},
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
				container: &statsContainer{},
				kvStore:   &kvstore.Memory{},
				logger:    model.DiscardLogger,
				mu:        sync.Mutex{},
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
						netxlite.NewStdlibResolver(log.Log),
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
func TestNewHTTPSDialerPolicy(t *testing.T) {
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
	}

	minimalUserPolicy := []byte(`{"Version":3}`)

	cases := []testcase{{
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
	}, {
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
	}, {
		name: "when there is no proxy URL and there is a user policy",
		kvStore: func() model.KeyValueStore {
			store := &kvstore.Memory{}
			// this policy is mostly empty but it's enough to load
			runtimex.Try0(store.Set(userPolicyKey, minimalUserPolicy))
			return store
		},
		proxyURL:   nil,
		expectType: "*enginenetx.userPolicy",
	}, {
		name: "when there is no proxy URL and there is no user policy",
		kvStore: func() model.KeyValueStore {
			return &kvstore.Memory{}
		},
		proxyURL:   nil,
		expectType: "*enginenetx.statsPolicy",
	}}

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
		})
	}
}
