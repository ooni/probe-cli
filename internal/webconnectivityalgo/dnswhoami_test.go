package webconnectivityalgo

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSWhoamiService(t *testing.T) {
	// expectation describes expectations
	type expectation struct {
		Entries []DNSWhoamiInfoEntry
		Good    bool
	}

	// testcase is a test case defined by this function
	type testcase struct {
		// name is the test case name
		name string

		// domain is the domain to query for
		domain string

		// expectations contains the expecations
		expectations []expectation
	}

	cases := []testcase{{
		name:   "common case using the default domain",
		domain: "", // forces using default
		expectations: []expectation{{
			Entries: []DNSWhoamiInfoEntry{{
				Address: netemx.DefaultClientAddress,
			}},
			Good: true,
		}, {
			Entries: []DNSWhoamiInfoEntry{{
				Address: netemx.DefaultClientAddress,
			}},
			Good: true,
		}},
	}, {
		name:   "error case using another domain",
		domain: "example.xyz",
		expectations: []expectation{{
			Entries: nil,
			Good:    false,
		}, {
			Entries: nil,
			Good:    false,
		}},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create testing scenario
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			// create the service
			svc := NewDNSWhoamiService(log.Log)

			// override fields
			svc.netx = &netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack}}
			if tc.domain != "" {
				svc.whoamiDomain = tc.domain
			}

			// prepare collecting results
			var results []expectation

			// run with the system resolver
			sysEntries, sysGood := svc.SystemV4(context.Background())
			results = append(results, expectation{
				Entries: sysEntries,
				Good:    sysGood,
			})

			// run with an UDP resolver
			udpEntries, udpGood := svc.UDPv4(context.Background(), "8.8.8.8:53")
			results = append(results, expectation{
				Entries: udpEntries,
				Good:    udpGood,
			})

			// check whether we've got what we expected
			if diff := cmp.Diff(tc.expectations, results); diff != "" {
				t.Fatal(diff)
			}
		})
	}

}
