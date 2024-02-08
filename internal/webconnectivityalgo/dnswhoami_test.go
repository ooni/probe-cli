package webconnectivityalgo

import (
	"context"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSWhoamiService(t *testing.T) {
	// callResults contains the results of calling System or UDPv4
	type callResults struct {
		Entries []DNSWhoamiInfoEntry
		Good    bool
	}

	// testcase is a test case defined by this function
	type testcase struct {
		// name is the test case name
		name string

		// domain is the domain to query for
		domain string

		// internals contains the expected internals cache
		internals map[string]*dnsWhoamiInfoTimedEntry

		// callResults contains the expectations
		callResults []callResults
	}

	cases := []testcase{{
		name:   "common case using the default domain",
		domain: "", // forces using default
		callResults: []callResults{{
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
		internals: map[string]*dnsWhoamiInfoTimedEntry{
			"system:///": {
				Addr: netemx.DefaultClientAddress,
				T:    time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC).Add(time.Second),
			},
			"8.8.8.8:53": {
				Addr: netemx.DefaultClientAddress,
				T:    time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC).Add(2 * time.Second),
			},
		},
	}, {
		name:   "error case using another domain",
		domain: "example.xyz",
		callResults: []callResults{{
			Entries: nil,
			Good:    false,
		}, {
			Entries: nil,
			Good:    false,
		}},
		internals: map[string]*dnsWhoamiInfoTimedEntry{},
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
			svc.timeNow = (&testTimeProvider{
				t0: time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC),
				times: []time.Duration{
					time.Second,
					2 * time.Second,
				},
				idx: 0,
			}).timeNow

			// prepare collecting results
			var results []callResults

			// run with the system resolver
			sysEntries, sysGood := svc.SystemV4(context.Background())
			results = append(results, callResults{
				Entries: sysEntries,
				Good:    sysGood,
			})

			// run with an UDP resolver
			udpEntries, udpGood := svc.UDPv4(context.Background(), "8.8.8.8:53")
			results = append(results, callResults{
				Entries: udpEntries,
				Good:    udpGood,
			})

			// check whether we've got what we expected
			if diff := cmp.Diff(tc.callResults, results); diff != "" {
				t.Fatal(diff)
			}

			// check the internals
			if diff := cmp.Diff(tc.internals, svc.cloneEntries()); diff != "" {
				t.Fatal(diff)
			}
		})
	}

	t.Run("we correctly handle cache expiration", func(t *testing.T) {
		// create testing scenario
		env := netemx.MustNewScenario(netemx.InternetScenario)
		defer env.Close()

		// create the service
		svc := NewDNSWhoamiService(log.Log)

		// create the timeTestProvider
		ttp := &testTimeProvider{
			t0: time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC),
			times: []time.Duration{
				// first run
				time.Second,
				2 * time.Second,
				// second run
				15 * time.Second,
				17 * time.Second,
				// third run
				60 * time.Second,
				62 * time.Second,
			},
			idx: 0,
		}

		// override fields
		svc.netx = &netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack}}
		svc.timeNow = ttp.timeNow

		// run for the first time
		_, _ = svc.SystemV4(context.Background())
		_, _ = svc.UDPv4(context.Background(), "8.8.8.8:53")

		// establish expectations for first run
		//
		// we expect the internals to be related to the first run
		expectFirstInternals := map[string]*dnsWhoamiInfoTimedEntry{
			"system:///": {
				Addr: netemx.DefaultClientAddress,
				T:    time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC).Add(time.Second),
			},
			"8.8.8.8:53": {
				Addr: netemx.DefaultClientAddress,
				T:    time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC).Add(2 * time.Second),
			},
		}

		// check the internals for the first run
		if diff := cmp.Diff(expectFirstInternals, svc.cloneEntries()); diff != "" {
			t.Fatal(diff)
		}

		// run for the second time
		_, _ = svc.SystemV4(context.Background())
		_, _ = svc.UDPv4(context.Background(), "8.8.8.8:53")

		// establish expectations for second run
		//
		// we expect the internals to be related to the first run because not
		// enough time has elapsed since we create the cache entries
		expectSecondInternals := map[string]*dnsWhoamiInfoTimedEntry{
			"system:///": {
				Addr: netemx.DefaultClientAddress,
				T:    time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC).Add(time.Second),
			},
			"8.8.8.8:53": {
				Addr: netemx.DefaultClientAddress,
				T:    time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC).Add(2 * time.Second),
			},
		}

		// check the internals for the second run
		if diff := cmp.Diff(expectSecondInternals, svc.cloneEntries()); diff != "" {
			t.Fatal(diff)
		}

		// run for the third time
		_, _ = svc.SystemV4(context.Background())
		_, _ = svc.UDPv4(context.Background(), "8.8.8.8:53")

		// establish expectations for third run
		//
		// we expect the cache to be related to the third run because now the
		// entries are stale and so we perform another lookup
		expectThirdInternals := map[string]*dnsWhoamiInfoTimedEntry{
			"system:///": {
				Addr: netemx.DefaultClientAddress,
				T:    time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC).Add(60 * time.Second),
			},
			"8.8.8.8:53": {
				Addr: netemx.DefaultClientAddress,
				T:    time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC).Add(62 * time.Second),
			},
		}

		// check the internals for the second run
		if diff := cmp.Diff(expectThirdInternals, svc.cloneEntries()); diff != "" {
			t.Fatal(diff)
		}
	})
}
