package enginenetx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestCircoPolicy(t *testing.T) {
	// define the config we're using in this test case
	circo := &CircoConfig{
		Beacons: map[string]CircoBeaconsDomain{
			"api.ooni.io": {
				IPAddrs: []string{
					"142.250.180.174",
					"2a00:1450:4002:809::200e",
				},
				SNIs: []string{
					"www.youtube.com",
				},
			},
		},
		Version: 0,
	}

	t.Run("Parallelism", func(t *testing.T) {
		const expected = 16
		p := &CircoPolicy{circo}
		if got := p.Parallelism(); got != expected {
			t.Fatal("wrong parallelism: expected", expected, "got", got)
		}
	})

	t.Run("LookupTactics", func(t *testing.T) {
		// testcase is a testcase for this function
		type testcase struct {
			// name is the test case name
			name string

			// domain is the domain we're looking up tactics for
			domain string

			// reso is the underlying resolver to use
			reso model.Resolver

			// expectErr is the error string we expect to see
			expectErr string

			// tactics contains the returned tactics
			tactics []HTTPSDialerTactic
		}

		cases := []testcase{

			// When the DNS fails and there is no beacon, there's nothing to do
			{
				name:   "with lookup host failure and non-beacon domain",
				domain: "www.example.com",
				reso: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, errors.New("dns_nxdomain_error")
					},
				},
				expectErr: "dns_nxdomain_error",
				tactics:   nil,
			},

			// When the DNS succeeds with bogons and there is no beacon available
			{
				name:   "with lookup host with bogons and non-beacon domain",
				domain: "www.example.com",
				reso: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						addrs := []string{
							"10.10.34.45",
						}
						return addrs, nil
					},
				},
				expectErr: "dns_no_answer",
				tactics:   nil,
			},

			// When the DNS succeeds and there is no beacon, we only schedule resolver addrs
			{
				name:   "with lookup host success and non-beacon domain",
				domain: "www.example.com",
				reso: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						addrs := []string{
							"2606:2800:220:1:248:1893:25c8:1946",
							"93.184.216.34",
						}
						return addrs, nil
					},
				},
				expectErr: "",
				tactics: []HTTPSDialerTactic{
					&circoTactic{
						Address:            "2606:2800:220:1:248:1893:25c8:1946",
						InitialWaitTime:    0,
						TLSServerName:      "www.example.com",
						X509VerifyHostname: "www.example.com",
					},
					&circoTactic{
						Address:            "93.184.216.34",
						InitialWaitTime:    300 * time.Millisecond,
						TLSServerName:      "www.example.com",
						X509VerifyHostname: "www.example.com",
					},
				},
			},

			// When the DNS fails and we have beacons, we're going to try using beacons
			// IP addresses starting from the normal SNI and then trying alternative ones
			{
				name:   "with lookup-host failure and beacon domain",
				domain: "api.ooni.io",
				reso: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, errors.New("dns_nxdomain_error")
					},
				},
				expectErr: "",
				tactics: []HTTPSDialerTactic{
					&circoTactic{
						Address:            "142.250.180.174",
						InitialWaitTime:    0,
						TLSServerName:      "api.ooni.io",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "2a00:1450:4002:809::200e",
						InitialWaitTime:    300 * time.Millisecond,
						TLSServerName:      "api.ooni.io",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "142.250.180.174",
						InitialWaitTime:    3 * time.Second,
						TLSServerName:      "www.youtube.com",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "2a00:1450:4002:809::200e",
						InitialWaitTime:    3300 * time.Millisecond,
						TLSServerName:      "www.youtube.com",
						X509VerifyHostname: "api.ooni.io",
					},
				},
			},

			// When the DNS succeeds and we have beacons, we eventually try using
			// beacons domains but after a quite large delay
			{
				name:   "with lookup host success and beacon domain",
				domain: "api.ooni.io",
				reso: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						addrs := []string{"162.55.247.208"}
						return addrs, nil
					},
				},
				expectErr: "",
				tactics: []HTTPSDialerTactic{
					&circoTactic{
						Address:            "142.250.180.174",
						InitialWaitTime:    0,
						TLSServerName:      "api.ooni.io",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "162.55.247.208",
						InitialWaitTime:    300 * time.Millisecond,
						TLSServerName:      "api.ooni.io",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "2a00:1450:4002:809::200e",
						InitialWaitTime:    600 * time.Millisecond,
						TLSServerName:      "api.ooni.io",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "142.250.180.174",
						InitialWaitTime:    3000 * time.Millisecond,
						TLSServerName:      "www.youtube.com",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "162.55.247.208",
						InitialWaitTime:    3300 * time.Millisecond,
						TLSServerName:      "www.youtube.com",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "2a00:1450:4002:809::200e",
						InitialWaitTime:    3600 * time.Millisecond,
						TLSServerName:      "www.youtube.com",
						X509VerifyHostname: "api.ooni.io",
					},
				},
			},

			// The lookup returns bogons but this domain is a beacon
			{
				name:   "with lookup host with bogons and beacon domain",
				domain: "api.ooni.io",
				reso: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						addrs := []string{"10.10.34.35"}
						return addrs, nil
					},
				},
				expectErr: "",
				tactics: []HTTPSDialerTactic{
					&circoTactic{
						Address:            "142.250.180.174",
						InitialWaitTime:    0,
						TLSServerName:      "api.ooni.io",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "2a00:1450:4002:809::200e",
						InitialWaitTime:    300 * time.Millisecond,
						TLSServerName:      "api.ooni.io",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "142.250.180.174",
						InitialWaitTime:    3000 * time.Millisecond,
						TLSServerName:      "www.youtube.com",
						X509VerifyHostname: "api.ooni.io",
					},
					&circoTactic{
						Address:            "2a00:1450:4002:809::200e",
						InitialWaitTime:    3300 * time.Millisecond,
						TLSServerName:      "www.youtube.com",
						X509VerifyHostname: "api.ooni.io",
					},
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				p := &CircoPolicy{circo}
				ctx := context.Background()
				tactics, err := p.LookupTactics(ctx, tc.domain, tc.reso)

				t.Logf("%s %s", tactics, err)

				// make sure the error is the expected one
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

				if diff := cmp.Diff(tc.tactics, tactics); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}
