package enginenetx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
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

// httpsDialerCancelingContextStatsTracker is an [HTTPSDialerStatsTracker] with a cancel
// function that causes the context to be canceled once we start dialing.
//
// This struct helps with testing [HTTPSDialer] is WAI when the context
// has been canceled and we correctly shutdown all goroutines.
type httpsDialerCancelingContextStatsTracker struct {
	cancel context.CancelFunc
	flags  int
}

var _ HTTPSDialerStatsTracker = &httpsDialerCancelingContextStatsTracker{}

// OnStarting implements HTTPSDialerStatsTracker.
func (st *httpsDialerCancelingContextStatsTracker) OnStarting(tactic *HTTPSDialerTactic) {
	if (st.flags & httpsDialerCancelingContextStatsTrackerOnStarting) != 0 {
		st.cancel()
	}
}

// OnTCPConnectError implements HTTPSDialerStatsTracker.
func (*httpsDialerCancelingContextStatsTracker) OnTCPConnectError(ctx context.Context, tactic *HTTPSDialerTactic, err error) {
	// nothing
}

// OnTLSHandshakeError implements HTTPSDialerStatsTracker.
func (*httpsDialerCancelingContextStatsTracker) OnTLSHandshakeError(ctx context.Context, tactic *HTTPSDialerTactic, err error) {
	// nothing
}

// OnTLSVerifyError implements HTTPSDialerStatsTracker.
func (*httpsDialerCancelingContextStatsTracker) OnTLSVerifyError(tactic *HTTPSDialerTactic, err error) {
	// nothing
}

// OnSuccess implements HTTPSDialerStatsTracker.
func (st *httpsDialerCancelingContextStatsTracker) OnSuccess(tactic *HTTPSDialerTactic) {
	if (st.flags & httpsDialerCancelingContextStatsTrackerOnSuccess) != 0 {
		st.cancel()
	}
}

func TestHTTPSDialerNetemQA(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the name of the test case
		name string

		// short indicates whether this is a short test
		short bool

		// stats is the stats tracker to use.
		stats HTTPSDialerStatsTracker

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
			stats:    &HTTPSDialerNullStatsTracker{},
			endpoint: "www.example.com", // note: here the port is missing
			scenario: netemx.InternetScenario,
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "address www.example.com: missing port in address",
		},

		// This test case ensures that we handle the case of a nonexistent domain
		// where we get a dns_no_answer error. The original DNS error is lost in
		// background goroutines and what we report to the caller is just that there
		// is no available IP address and tactic to attempt using.
		{
			name:     "hd.policy.LookupTactics failure",
			short:    true,
			stats:    &HTTPSDialerNullStatsTracker{},
			endpoint: "www.example.nonexistent:443", // note: the domain does not exist
			scenario: netemx.InternetScenario,
			configureDPI: func(dpi *netem.DPIEngine) {
				// nothing
			},
			expectErr: "dns_no_answer",
		},

		// This test case is the common case: all is good with multiple addresses to dial (I am
		// not testing the case of a single address because it's a subcase of this one)
		{
			name:     "successful dial with multiple addresses",
			short:    true,
			stats:    &HTTPSDialerNullStatsTracker{},
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
			stats:    &HTTPSDialerNullStatsTracker{},
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
			stats:    &HTTPSDialerNullStatsTracker{},
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
			stats:    &HTTPSDialerNullStatsTracker{},
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
			stats:    &HTTPSDialerNullStatsTracker{},
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
			stats:    &HTTPSDialerNullStatsTracker{},
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
			name:  "with context being canceled in OnStarting",
			short: true,
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
			expectErr: "interrupted\ninterrupted",
		},

		// This is another corner case: what happens if the context is canceled
		// right after we eastablish a connection? Because of how the current code
		// is written, the easiest thing to do is to just return the conn.
		{
			name:  "with context being canceled in OnSuccess for the first success",
			short: true,
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

				policy := &dnsPolicy{
					Logger:   log.Log,
					Resolver: resolver,
				}

				// create the TLS dialer
				dialer := NewHTTPSDialer(
					log.Log,
					netx,
					policy,
					tc.stats,
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
			}()

			// now verify that we have closed all the connections
			if err := cv.CheckForOpenConns(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHTTPSDialerTactic(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		expected := `{"Address":"162.55.247.208","InitialDelay":150000000,"Port":"443","SNI":"www.example.com","VerifyHostname":"api.ooni.io"}`
		ldt := &HTTPSDialerTactic{
			Address:        "162.55.247.208",
			InitialDelay:   150 * time.Millisecond,
			Port:           "443",
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
		var expect HTTPSDialerTactic
		ff.Fill(&expect)
		got := expect.Clone()
		if diff := cmp.Diff(expect.String(), got.String()); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Summary", func(t *testing.T) {
		expected := `162.55.247.208:443 sni=www.example.com verify=api.ooni.io`
		ldt := &HTTPSDialerTactic{
			Address:        "162.55.247.208",
			InitialDelay:   150 * time.Millisecond,
			Port:           "443",
			SNI:            "www.example.com",
			VerifyHostname: "api.ooni.io",
		}
		got := ldt.Summary()
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestHTTPSDialerHostNetworkQA(t *testing.T) {
	t.Run("HTTPSDialerNullPolicy allows connecting to https://127.0.0.1/ using a custom CA", func(t *testing.T) {
		ca := netem.MustNewCA()
		server := testingx.MustNewHTTPServerTLS(
			testingx.HTTPHandlerBlockpage451(),
			ca,
			"server.local",
		)
		defer server.Close()

		tproxy := &netxlite.DefaultTProxy{}

		// The resolver we're creating here reproduces the test case described by
		// https://github.com/ooni/probe-cli/pull/1295#issuecomment-1731243994
		resolver := netxlite.MaybeWrapWithBogonResolver(true, netxlite.NewStdlibResolver(log.Log))

		httpsDialer := NewHTTPSDialer(
			log.Log,
			&netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
				MockDefaultCertPool: func() *x509.CertPool {
					return ca.DefaultCertPool() // just override the CA
				},
				MockDialTimeout:                tproxy.DialTimeout,
				MockDialContext:                tproxy.DialContext,
				MockListenTCP:                  tproxy.ListenTCP,
				MockListenUDP:                  tproxy.ListenUDP,
				MockGetaddrinfoLookupANY:       tproxy.GetaddrinfoLookupANY,
				MockGetaddrinfoResolverNetwork: tproxy.GetaddrinfoResolverNetwork,
			}},
			&dnsPolicy{
				Logger:   log.Log,
				Resolver: resolver,
			},
			&HTTPSDialerNullStatsTracker{},
		)

		URL := runtimex.Try1(url.Parse(server.URL))

		ctx := context.Background()
		tlsConn, err := httpsDialer.DialTLSContext(ctx, "tcp", URL.Host)
		if err != nil {
			t.Fatal(err)
		}
		tlsConn.Close()
	})
}

func TestHTTPSDialerVerifyCertificateChain(t *testing.T) {
	t.Run("without any peer certificate", func(t *testing.T) {
		tlsConn := &mocks.TLSConn{
			MockConnectionState: func() tls.ConnectionState {
				return tls.ConnectionState{} // empty!
			},
		}
		certPool := netxlite.NewMozillaCertPool()
		err := httpsDialerVerifyCertificateChain("www.example.com", tlsConn, certPool)
		if !errors.Is(err, errNoPeerCertificate) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with an empty hostname", func(t *testing.T) {
		tlsConn := &mocks.TLSConn{
			MockConnectionState: func() tls.ConnectionState {
				return tls.ConnectionState{} // empty but should not be an issue
			},
		}
		certPool := netxlite.NewMozillaCertPool()
		err := httpsDialerVerifyCertificateChain("", tlsConn, certPool)
		if !errors.Is(err, errEmptyVerifyHostname) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestHTTPSDialerReduceResult(t *testing.T) {
	t.Run("we return the first conn in a list of conns and close the other conns", func(t *testing.T) {
		var closed int
		expect := &mocks.TLSConn{} // empty
		connv := []model.TLSConn{
			expect,
			&mocks.TLSConn{
				Conn: mocks.Conn{
					MockClose: func() error {
						closed++
						return nil
					},
				},
			},
			&mocks.TLSConn{
				Conn: mocks.Conn{
					MockClose: func() error {
						closed++
						return nil
					},
				},
			},
		}

		conn, err := httpsDialerReduceResult(connv, nil)
		if err != nil {
			t.Fatal(err)
		}

		if conn != expect {
			t.Fatal("unexpected conn")
		}

		if closed != 2 {
			t.Fatal("did not call close")
		}
	})

	t.Run("we join together a list of errors", func(t *testing.T) {
		expectErr := "connection_refused\ninterrupted"
		errorv := []error{errors.New("connection_refused"), errors.New("interrupted")}

		conn, err := httpsDialerReduceResult(nil, errorv)
		if err == nil || err.Error() != expectErr {
			t.Fatal("unexpected err", err)
		}

		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("with a single error we return such an error", func(t *testing.T) {
		expected := errors.New("connection_refused")
		errorv := []error{expected}

		conn, err := httpsDialerReduceResult(nil, errorv)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}

		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("we return errDNSNoAnswer if we don't have any conns or errors to return", func(t *testing.T) {
		conn, err := httpsDialerReduceResult(nil, nil)
		if !errors.Is(err, errDNSNoAnswer) {
			t.Fatal("unexpected error", err)
		}

		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})
}
