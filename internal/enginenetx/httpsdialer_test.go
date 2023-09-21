package enginenetx_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/enginenetx"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// Flags controlling when [httpsDialerCancelingContextStatsTracker] cancels the context
const (
	httpsDialerCancelingContextStatsTrackerOnStarting = 1 << iota
	httpsDialerCancelingContextStatsTrackerOnSuccess
)

// httpsDialerCancelingContextStatsTracker is an [enginenetx.HTTPSDialerStatsTracker] with a cancel
// function that causes the context to be canceled once we start dialing.
//
// This struct helps with testing [enginenetx.HTTPSDialer] is WAI when the context
// has been canceled and we correctly shutdown all goroutines.
type httpsDialerCancelingContextStatsTracker struct {
	cancel context.CancelFunc
	flags  int
}

var _ enginenetx.HTTPSDialerStatsTracker = &httpsDialerCancelingContextStatsTracker{}

// OnStarting implements enginenetx.HTTPSDialerStatsTracker.
func (st *httpsDialerCancelingContextStatsTracker) OnStarting(tactic *enginenetx.HTTPSDialerTactic) {
	if (st.flags & httpsDialerCancelingContextStatsTrackerOnStarting) != 0 {
		st.cancel()
	}
}

// OnTCPConnectError implements enginenetx.HTTPSDialerStatsTracker.
func (*httpsDialerCancelingContextStatsTracker) OnTCPConnectError(ctx context.Context, tactic *enginenetx.HTTPSDialerTactic, err error) {
	// nothing
}

// OnTLSHandshakeError implements enginenetx.HTTPSDialerStatsTracker.
func (*httpsDialerCancelingContextStatsTracker) OnTLSHandshakeError(ctx context.Context, tactic *enginenetx.HTTPSDialerTactic, err error) {
	// nothing
}

// OnTLSVerifyError implements enginenetx.HTTPSDialerStatsTracker.
func (*httpsDialerCancelingContextStatsTracker) OnTLSVerifyError(ctz context.Context, tactic *enginenetx.HTTPSDialerTactic, err error) {
	// nothing
}

// OnSuccess implements enginenetx.HTTPSDialerStatsTracker.
func (st *httpsDialerCancelingContextStatsTracker) OnSuccess(tactic *enginenetx.HTTPSDialerTactic) {
	if (st.flags & httpsDialerCancelingContextStatsTrackerOnSuccess) != 0 {
		st.cancel()
	}
}

func TestHTTPSDialerWAI(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the name of the test case
		name string

		// short indicates whether this is a short test
		short bool

		// policy is the dialer policy
		policy enginenetx.HTTPSDialerPolicy

		// stats is the stats tracker to use.
		stats enginenetx.HTTPSDialerStatsTracker

		// endpoint is the endpoint to connect to consisting of a domain
		// name or IP address followed by a TCP port
		endpoint string

		// scenario is the netemx testing scenario to create
		scenario []*netemx.ScenarioDomainAddresses

		// configureDPI configures DPI rules (just add an empty
		// function if you don't need any)
		configureDPI func(dpi *netem.DPIEngine)

		// expectErr is the error string we expect to see
		expectErr string
	}

	allTestCases := []testcase{

		// This test case ensures that we handle the corner case of a missing port
		{
			name:     "net.SplitHostPort failure",
			short:    true,
			policy:   &enginenetx.HTTPSDialerNullPolicy{},
			stats:    &enginenetx.HTTPSDialerNullStatsTracker{},
			endpoint: "www.example.com", // note: here the port is missing
			scenario: netemx.InternetScenario,
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "address www.example.com: missing port in address",
		},

		// This test case ensures that we handle the case of a nonexistent domain
		{
			name:     "hd.policy.LookupTactics failure",
			short:    true,
			policy:   &enginenetx.HTTPSDialerNullPolicy{},
			stats:    &enginenetx.HTTPSDialerNullStatsTracker{},
			endpoint: "www.example.nonexistent:443", // note: the domain does not exist
			scenario: netemx.InternetScenario,
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "dns_nxdomain_error",
		},

		// This test case is the common case: all is good with multiple addresses to dial (I am
		// not testing the case of a single address because it's a subcase of this one)
		{
			name:     "successful dial with multiple addresses",
			short:    true,
			policy:   &enginenetx.HTTPSDialerNullPolicy{},
			stats:    &enginenetx.HTTPSDialerNullStatsTracker{},
			endpoint: "www.example.com:443",
			scenario: []*netemx.ScenarioDomainAddresses{{
				Domains: []string{
					"www.example.com",
				},
				Addresses: []string{
					"93.184.216.34",
					"93.184.216.35",
					"93.184.216.36",
					"93.184.216.37",
				},
				Role:             netemx.ScenarioRoleWebServer,
				ServerNameMain:   "www.example.com",
				WebServerFactory: netemx.ExampleWebPageHandlerFactory(),
			}},
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "",
		},

		// Here we make sure that we're doing OK if the addresses are TCP-blocked
		{
			name:     "with TCP connect errors",
			short:    true,
			policy:   &enginenetx.HTTPSDialerNullPolicy{},
			stats:    &enginenetx.HTTPSDialerNullStatsTracker{},
			endpoint: "www.example.com:443",
			scenario: []*netemx.ScenarioDomainAddresses{{
				Domains: []string{
					"www.example.com",
				},
				Addresses: []string{
					"93.184.216.34",
					"93.184.216.35",
				},
				Role:             netemx.ScenarioRoleWebServer,
				ServerNameMain:   "www.example.com",
				WebServerFactory: netemx.ExampleWebPageHandlerFactory(),
			}},
			configureDPI: func(dpi *netem.DPIEngine) {
				// we force closing the connection for all the known server endpoints
				dpi.AddRule(&netem.DPICloseConnectionForServerEndpoint{
					Logger:          log.Log,
					ServerIPAddress: "93.184.216.34",
					ServerPort:      443,
				})
				dpi.AddRule(&netem.DPICloseConnectionForServerEndpoint{
					Logger:          log.Log,
					ServerIPAddress: "93.184.216.35",
					ServerPort:      443,
				})
			},
			expectErr: "connection_refused\nconnection_refused",
		},

		// Here we're making sure it's all WAI when there is TLS interference
		{
			name:     "with TLS handshake errors",
			short:    true,
			policy:   &enginenetx.HTTPSDialerNullPolicy{},
			stats:    &enginenetx.HTTPSDialerNullStatsTracker{},
			endpoint: "www.example.com:443",
			scenario: []*netemx.ScenarioDomainAddresses{{
				Domains: []string{
					"www.example.com",
				},
				Addresses: []string{
					"93.184.216.34",
					"93.184.216.35",
				},
				Role:             netemx.ScenarioRoleWebServer,
				ServerNameMain:   "www.example.com",
				WebServerFactory: netemx.ExampleWebPageHandlerFactory(),
			}},
			configureDPI: func(dpi *netem.DPIEngine) {
				// we force resetting the connection for www.example.com
				dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
					Logger: log.Log,
					SNI:    "www.example.com",
				})
			},
			expectErr: "connection_reset\nconnection_reset",
		},

		// Note: this is where we test that TLS verification is WAI. The netemx scenario role
		// constructs the equivalent of real world's badssl.com and we're checking whether
		// we would accept a certificate valid for another hostname. The answer should be "NO!".
		{
			name:     "with a TLS certificate valid for ANOTHER domain",
			short:    true,
			policy:   &enginenetx.HTTPSDialerNullPolicy{},
			stats:    &enginenetx.HTTPSDialerNullStatsTracker{},
			endpoint: "wrong.host.badssl.com:443",
			scenario: []*netemx.ScenarioDomainAddresses{{
				Domains: []string{
					"wrong.host.badssl.com",
					"untrusted-root.badssl.com",
					"expired.badssl.com",
				},
				Addresses: []string{
					"93.184.216.34",
					"93.184.216.35",
				},
				Role: netemx.ScenarioRoleBadSSL,
			}},
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "ssl_invalid_hostname\nssl_invalid_hostname",
		},

		// Note: this is another TLS related test case where we make sure that
		// we can handle an untrusted root/self signed certificate
		{
			name:     "with TLS certificate signed by an unknown authority",
			short:    true,
			policy:   &enginenetx.HTTPSDialerNullPolicy{},
			stats:    &enginenetx.HTTPSDialerNullStatsTracker{},
			endpoint: "untrusted-root.badssl.com:443",
			scenario: []*netemx.ScenarioDomainAddresses{{
				Domains: []string{
					"wrong.host.badssl.com",
					"untrusted-root.badssl.com",
					"expired.badssl.com",
				},
				Addresses: []string{
					"93.184.216.34",
					"93.184.216.35",
				},
				Role: netemx.ScenarioRoleBadSSL,
			}},
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "ssl_unknown_authority\nssl_unknown_authority",
		},

		// Note: this is another TLS related test case where we make sure that
		// we can handle a certificate that has now expired.
		{
			name:     "with expired TLS certificate",
			short:    true,
			policy:   &enginenetx.HTTPSDialerNullPolicy{},
			stats:    &enginenetx.HTTPSDialerNullStatsTracker{},
			endpoint: "expired.badssl.com:443",
			scenario: []*netemx.ScenarioDomainAddresses{{
				Domains: []string{
					"wrong.host.badssl.com",
					"untrusted-root.badssl.com",
					"expired.badssl.com",
				},
				Addresses: []string{
					"93.184.216.34",
					"93.184.216.35",
				},
				Role: netemx.ScenarioRoleBadSSL,
			}},
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "ssl_invalid_certificate\nssl_invalid_certificate",
		},

		// This is a corner case: what if the context is canceled after the DNS lookup
		// but before we start dialing? Are we closing all goroutines and returning correctly?
		{
			name:   "with context being canceled in OnStarting",
			short:  true,
			policy: &enginenetx.HTTPSDialerNullPolicy{},
			stats: &httpsDialerCancelingContextStatsTracker{
				cancel: nil,
				flags:  httpsDialerCancelingContextStatsTrackerOnStarting,
			},
			endpoint: "www.example.com:443",
			scenario: []*netemx.ScenarioDomainAddresses{{
				Domains: []string{
					"www.example.com",
				},
				Addresses: []string{
					"93.184.216.34",
					"93.184.216.35",
				},
				Role:             netemx.ScenarioRoleWebServer,
				ServerNameMain:   "www.example.com",
				WebServerFactory: netemx.ExampleWebPageHandlerFactory(),
			}},
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "context canceled",
		},

		// This is another corner case: what happens if the context is canceled after we
		// have a good connection but before we're able to report it to the caller?
		{
			name:   "with context being canceled in OnSuccess for the first success",
			short:  true,
			policy: &enginenetx.HTTPSDialerNullPolicy{},
			stats: &httpsDialerCancelingContextStatsTracker{
				cancel: nil,
				flags:  httpsDialerCancelingContextStatsTrackerOnSuccess,
			},
			endpoint: "www.example.com:443",
			scenario: []*netemx.ScenarioDomainAddresses{{
				Domains: []string{
					"www.example.com",
				},
				Addresses: []string{
					"93.184.216.34",
					"93.184.216.35",
				},
				Role:             netemx.ScenarioRoleWebServer,
				ServerNameMain:   "www.example.com",
				WebServerFactory: netemx.ExampleWebPageHandlerFactory(),
			}},
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "context canceled",
		}}

	for _, tc := range allTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// make sure we honor `go test -short`
			if !tc.short && testing.Short() {
				t.Skip("skip test in short mode")
			}

			// track all the connections so we can check whether we close them all
			cv := &testingx.CloseVerify{}

			func() {
				// create the QA environment
				env := netemx.MustNewScenario(tc.scenario)
				defer env.Close()

				// possibly add specific DPI rules
				tc.configureDPI(env.DPIEngine())

				// create the proper underlying network and wrap it such that
				// we track whether we close all the connections
				unet := cv.WrapUnderlyingNetwork(&netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack})

				// create the network proper
				netx := &netxlite.Netx{Underlying: unet}

				// create the getaddrinfo resolver
				resolver := netx.NewStdlibResolver(log.Log)

				// create the TLS dialer
				dialer := enginenetx.NewHTTPSDialer(
					log.Log,
					tc.policy,
					resolver,
					tc.stats,
					unet,
				)
				defer dialer.CloseIdleConnections()

				// configure cancellable context--some tests are going to use cancel
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Possibly tell the httpsDialerCancelingContextStatsTracker about the cancel func
				// depending on which flags have been configured.
				if p, ok := tc.stats.(*httpsDialerCancelingContextStatsTracker); ok {
					p.cancel = cancel
				}

				// dial the TLS connection
				tlsConn, err := dialer.DialTLSContext(ctx, "tcp", tc.endpoint)

				t.Logf("%+v %+v", tlsConn, err)

				// make sure the error is the one we expected
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

				// make sure we close the conn
				if tlsConn != nil {
					defer tlsConn.Close()
				}

				// wait for background connections to join
				dialer.WaitGroup().Wait()
			}()

			// now verify that we have closed all the connections
			if err := cv.CheckForOpenConns(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLoadHTTPSDialerPolicy(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the test case name
		name string

		// input contains the serialized input bytes
		input []byte

		// expectErr contains the expected error string or the empty string on success
		expectErr string

		// expectPolicy contains the expected policy we loaded or nil
		expectedPolicy *enginenetx.HTTPSDialerLoadablePolicy
	}

	cases := []testcase{{
		name:           "with nil input",
		input:          nil,
		expectErr:      "unexpected end of JSON input",
		expectedPolicy: nil,
	}, {
		name:           "with invalid serialized JSON",
		input:          []byte(`{`),
		expectErr:      "unexpected end of JSON input",
		expectedPolicy: nil,
	}, {
		name:           "with empty serialized JSON",
		input:          []byte(`{}`),
		expectErr:      "",
		expectedPolicy: &enginenetx.HTTPSDialerLoadablePolicy{},
	}, {
		name: "with real serialized policy",
		input: (func() []byte {
			return runtimex.Try1(json.Marshal(&enginenetx.HTTPSDialerLoadablePolicy{
				Domains: map[string][]*enginenetx.HTTPSDialerTactic{
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
			}))
		})(),
		expectErr: "",
		expectedPolicy: &enginenetx.HTTPSDialerLoadablePolicy{
			Domains: map[string][]*enginenetx.HTTPSDialerTactic{
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
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := enginenetx.LoadHTTPSDialerPolicy(tc.input)

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
}

func TestHTTPSDialerTactic(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		expected := `{"Endpoint":"162.55.247.208:443","InitialDelay":150000000,"SNI":"www.example.com","VerifyHostname":"api.ooni.io"}`
		ldt := &enginenetx.HTTPSDialerTactic{
			Endpoint:       "162.55.247.208:443",
			InitialDelay:   150 * time.Millisecond,
			SNI:            "www.example.com",
			VerifyHostname: "api.ooni.io",
		}
		got := ldt.String()
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Clone", func(t *testing.T) {
		ff := &testingx.FakeFiller{}
		var expect enginenetx.HTTPSDialerTactic
		ff.Fill(&expect)
		got := expect.Clone()
		if diff := cmp.Diff(expect.String(), got.String()); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Summary", func(t *testing.T) {
		expected := `162.55.247.208:443 sni=www.example.com verify=api.ooni.io`
		ldt := &enginenetx.HTTPSDialerTactic{
			Endpoint:       "162.55.247.208:443",
			InitialDelay:   150 * time.Millisecond,
			SNI:            "www.example.com",
			VerifyHostname: "api.ooni.io",
		}
		got := ldt.Summary()
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	})
}
