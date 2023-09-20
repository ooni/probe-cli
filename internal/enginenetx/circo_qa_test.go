package enginenetx_test

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/enginenetx"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestCircoQA(t *testing.T) {
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

	allTestCases := []testcase{

		{
			name:  "successful case",
			short: true,
			policy: &enginenetx.CircoPolicy{
				Config: enginenetx.NewCircoConfig(),
			},
			endpoint: "api.ooni.io:443",
			scenario: netemx.InternetScenario,
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "",
		},

		{
			name:  "with just SNI based blocking",
			short: true,
			policy: &enginenetx.CircoPolicy{
				Config: enginenetx.NewCircoConfig(),
			},
			endpoint: "api.ooni.io:443",
			scenario: netemx.InternetScenario,
			configureDPI: func(dpi *netem.DPIEngine) {
				dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
					Logger: log.Log,
					SNI:    "api.ooni.io",
				})
			},
			expectErr: "",
		},

		{
			name:  "with DNS injection and SNI based blocking",
			short: true,
			policy: &enginenetx.CircoPolicy{
				Config: enginenetx.NewCircoConfig(),
			},
			endpoint: "api.ooni.io:443",
			scenario: netemx.InternetScenario,
			configureDPI: func(dpi *netem.DPIEngine) {
				dpi.AddRule(&netem.DPISpoofDNSResponse{
					Addresses: []string{
						netemx.AddressPublicBlockpage,
					},
					Logger: log.Log,
					Domain: "api.ooni.io",
				})
				dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
					Logger: log.Log,
					SNI:    "api.ooni.io",
				})
			},
			expectErr: "",
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
					unet,
				)
				defer dialer.CloseIdleConnections()

				// configure context
				ctx := context.Background()

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
