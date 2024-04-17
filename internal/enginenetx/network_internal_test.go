package enginenetx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	http "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/logmodel"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
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
				time.Now,
			)

			got := fmt.Sprintf("%T", p)
			if diff := cmp.Diff(tc.expectType, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

// The purpose of this function is to mock the dialTLSFn field to collect the tactics
// that would be used by the (*httpsDialer) to guarantee that we obtain the correct scheduling
// for the tactics in a variety of working conditions. For example, let us assume that the
// bridge is broken, then we want to know _when_ we're trying to use the DNS.
func TestNetworkVerifyGeneratedTactics(t *testing.T) {

	// testEnv is the environment for running a given test
	type testEnv struct {
		// hdx is the HTTPS dialer we're using
		hdx *httpsDialer

		// store is the kvstore we're using
		store *kvstore.Memory
	}

	// newPredictableTimeGenerator returns a predictable time generator.
	newPredictableTimeGenerator := func() func() time.Time {
		return testingx.NewTimeDeterministic(time.Date(2024, 4, 17, 14, 56, 0, 0, time.UTC)).Now
	}

	// newTestEnv creates a new [*testEnv] instance.
	newTestEnv := func(container *statsContainer, err error, addrs ...string) *testEnv {
		// create an in-memory key-value store
		store := &kvstore.Memory{}

		// serialize and write the stats container into the kvstore
		runtimex.Try0(store.Set(statsKey, must.MarshalJSON(container)))

		// create a fake resolver resolving the given addrs
		reso := &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return append([]string{}, addrs...), err
			},
			MockNetwork: func() string {
				return netxlite.StdlibResolverGetaddrinfo
			},
			MockAddress: func() string {
				return ""
			},
			MockCloseIdleConnections: func() {
				// nothing
			},
			MockLookupHTTPS: nil,
			MockLookupNS:    nil,
		}

		// use deterministic time to have predictable random shuffling
		timeNow := newPredictableTimeGenerator()

		// create the network and the HTTPS dialer
		network, hdx := newNetwork(
			bytecounter.New(),
			store,
			model.DiscardLogger,
			nil, // proxyURL disabled
			reso,
			timeNow,
		)

		// ignore the network
		_ = network

		// fill and return the result
		return &testEnv{
			hdx:   hdx,
			store: store,
		}
	}

	// obtainDialerTactics runs DialTLSContext with a mocked function
	// such that we can obtain the tactics that would be used.
	obtainDialerTactics := func(ctx context.Context, network, endpoint string, tex *testEnv) []*httpsDialerTactic {
		// arrange for extracting the tactics that would be used
		var (
			buffer []*httpsDialerTactic
			mu     sync.Mutex
		)
		tex.hdx.dialTLSFn = func(
			ctx context.Context,
			logger logmodel.Logger,
			t0 time.Time,
			tactic *httpsDialerTactic,
		) (http.TLSConn, error) {
			mu.Lock()
			buffer = append(buffer, tactic)
			mu.Unlock()
			return nil, errors.New("dialing disabled")
		}

		// perform the actual dial
		_, _ = tex.hdx.DialTLSContext(ctx, network, endpoint)

		// copy the tactics
		mu.Lock()
		output := append([]*httpsDialerTactic{}, buffer...)
		mu.Unlock()

		// return results
		return output
	}

	// generateExpectedBridgesSchedule generates the expected bridges schedule
	generateExpectedBridgesSchedule := func() []*httpsDialerTactic {
		output := []*httpsDialerTactic{}

		for _, ipaddr := range bridgesAddrs() {
			for _, domain := range bridgesDomainsInRandomOrder(newPredictableTimeGenerator()) {
			}
		}

		return output
	}

	// testCase is a test case for this function
	type testCase struct {
		// name is the test case name
		name string

		// dialEndpoint is the endpoint we should dial.
		dialEndpoint string

		// dnsAddrs are the addrs that the DNS should return
		dnsAddrs []string

		// dnsErr is the error that the DNS should return
		dnsErr error

		// initialStatsContainer is the initial stats container context
		initialStatsContainer *statsContainer

		// expectTactics contains the expected tactics
		expectTactics []*httpsDialerTactic
	}

	// cases contains all the test cases
	cases := []testCase{

		{
			name:                  "DNS=failing, cache=empty, domain=api.ooni.io",
			dialEndpoint:          "api.ooni.io:443",
			dnsAddrs:              []string{},
			dnsErr:                errors.New("dns_nxdomain_error"),
			initialStatsContainer: nil,
			expectTactics:         []*httpsDialerTactic{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create a new testing environment
			env := newTestEnv(tc.initialStatsContainer, tc.dnsErr, tc.dnsAddrs...)

			// create a background context
			ctx := context.Background()

			// obtain the tactics that would be used
			output := obtainDialerTactics(ctx, "tcp", tc.dialEndpoint, env)

			// compare with expectations
			if diff := cmp.Diff(tc.expectTactics, output); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
