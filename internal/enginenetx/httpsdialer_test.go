package enginenetx_test

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/enginenetx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Flags controlling when [httpsDialerPolicyCancelingContext] cancels the context
const (
	httpsDialerPolicyCancelingContextOnStarting = 1 << iota
	httpsDialerPolicyCancelingContextOnSuccess
)

// httpsDialerPolicyCancelingContext is an [enginenetsx.HTTPSDialerPolicy] with a cancel
// function that causes the context to be canceled once we start dialing.
//
// This struct helps with testing [enginenetx.HTTPSDialer] is WAI when the context
// has been canceled and we correctly shutdown all goroutines.
type httpsDialerPolicyCancelingContext struct {
	cancel context.CancelFunc
	flags  int
	policy enginenetx.HTTPSDialerPolicy
}

var _ enginenetx.HTTPSDialerPolicy = &httpsDialerPolicyCancelingContext{}

// LookupTactics implements enginenetx.HTTPSDialerPolicy.
func (p *httpsDialerPolicyCancelingContext) LookupTactics(ctx context.Context, domain string, reso model.Resolver) ([]enginenetx.HTTPSDialerTactic, error) {
	tactics, err := p.policy.LookupTactics(ctx, domain, reso)
	if err != nil {
		return nil, err
	}
	var out []enginenetx.HTTPSDialerTactic
	for _, tactic := range tactics {
		out = append(out, &httpsDialerTacticCancelingContext{
			HTTPSDialerTactic: tactic,
			cancel:            p.cancel,
			flags:             p.flags,
		})
	}
	return out, nil
}

// Parallelism implements enginenetx.HTTPSDialerPolicy.
func (p *httpsDialerPolicyCancelingContext) Parallelism() int {
	return p.policy.Parallelism()
}

// httpsDialerTacticCancelingContext is the tactic returned by [httpsDialerPolicyCancelingContext].
type httpsDialerTacticCancelingContext struct {
	enginenetx.HTTPSDialerTactic
	cancel context.CancelFunc
	flags  int
}

// OnStarting implements enginenetx.HTTPSDialerTactic.
func (t *httpsDialerTacticCancelingContext) OnStarting() {
	if (t.flags & httpsDialerPolicyCancelingContextOnStarting) != 0 {
		t.cancel()
	}
}

// OnSuccess implements enginenetx.HTTPSDialerTactic.
func (t *httpsDialerTacticCancelingContext) OnSuccess() {
	if (t.flags & httpsDialerPolicyCancelingContextOnSuccess) != 0 {
		t.cancel()
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

	allTestCases := []testcase{{
		name:     "net.SplitHostPort failure",
		short:    true,
		policy:   &enginenetx.HTTPSDialerNullPolicy{},
		endpoint: "www.example.com", // note: here the port is missing
		scenario: netemx.InternetScenario,
		configureDPI: func(dpi *netem.DPIEngine) {
			// nothing
		},
		expectErr: "address www.example.com: missing port in address",
	}, {
		name:     "hd.policy.LookupTactics failure",
		short:    true,
		policy:   &enginenetx.HTTPSDialerNullPolicy{},
		endpoint: "www.example.nonexistent:443", // note: the domain does not exist
		scenario: netemx.InternetScenario,
		configureDPI: func(dpi *netem.DPIEngine) {
			// nothing
		},
		expectErr: "dns_nxdomain_error",
	}, {
		name:     "successful dial with multiple addresses",
		short:    true,
		policy:   &enginenetx.HTTPSDialerNullPolicy{},
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
			WebServerFactory: netemx.ExampleWebPageHandlerFactory(),
		}},
		configureDPI: func(dpi *netem.DPIEngine) {
			// nothing
		},
		expectErr: "",
	}, {
		name:     "with TCP connect errors",
		short:    true,
		policy:   &enginenetx.HTTPSDialerNullPolicy{},
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
	}, {
		name:     "with TLS handshake errors",
		short:    true,
		policy:   &enginenetx.HTTPSDialerNullPolicy{},
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
	}, {
		// Note: this is where we test that TLS verification is WAI. The netemx scenario role
		// constructs the equivalent of real world's badssl.com and we're checking whether
		// we would accept a certificate valid for another hostname. The answer should be "NO!".
		name:     "with TLS verification errors",
		short:    true,
		policy:   &enginenetx.HTTPSDialerNullPolicy{},
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
	}, {
		name:  "with context being canceled in OnStarting",
		short: true,
		policy: &httpsDialerPolicyCancelingContext{
			cancel: nil,
			flags:  httpsDialerPolicyCancelingContextOnStarting,
			policy: &enginenetx.HTTPSDialerNullPolicy{},
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
			WebServerFactory: netemx.ExampleWebPageHandlerFactory(),
		}},
		configureDPI: func(dpi *netem.DPIEngine) {
			// nothing
		},
		expectErr: "context canceled",
	}, {
		name:  "with context being canceled in OnSuccess for the first success",
		short: true,
		policy: &httpsDialerPolicyCancelingContext{
			cancel: nil,
			flags:  httpsDialerPolicyCancelingContextOnSuccess,
			policy: &enginenetx.HTTPSDialerNullPolicy{},
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

			// create the QA environment
			env := netemx.MustNewScenario(tc.scenario)
			defer env.Close()

			// possibly add specific DPI rules
			tc.configureDPI(env.DPIEngine())

			// create the proper underlying network
			unet := &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack}

			// create the network proper
			netx := &netxlite.Netx{Underlying: unet}

			// create the getaddrinfo resolver
			resolver := netx.NewStdlibResolver(log.Log)

			// create the TLS dialer
			dialer := enginenetx.NewHTTPSDialer(
				log.Log,
				tc.policy,
				resolver,
				unet,
			)
			defer dialer.CloseIdleConnections()

			// configure cancellable context--some tests are going to use cancel
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Possibly tell the httpsDialerPolicyCancelingContext about the cancel func
			// depending on which flags have been configured.
			if p, ok := tc.policy.(*httpsDialerPolicyCancelingContext); ok {
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
		})
	}
}
