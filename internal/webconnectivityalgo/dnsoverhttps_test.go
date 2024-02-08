package webconnectivityalgo

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type testTimeProvider struct {
	t0    time.Time
	times []time.Duration
	idx   int
}

func (ttp *testTimeProvider) timeNow() time.Time {
	runtimex.Assert(ttp.idx < len(ttp.times), "out of bounds")
	mockedTime := ttp.t0.Add(ttp.times[ttp.idx])
	ttp.idx++
	return mockedTime
}

func TestOpportunisticDNSOverHTTPSURLProvider(t *testing.T) {

	// expectation is an expectation of a test case.
	type expectation struct {
		URL  string
		Good bool
	}

	// testcase is a test case implemented by this testing function.
	type testcase struct {
		// name is the test case name.
		name string

		// timeNow is the function to obtain time. In case it is zero, we're
		// not goint to reconfigure the time fetching function.
		timeNow func() time.Time

		// seed is the random seed or zero. In case it is zero, we're not
		// going to reconfigure the random see we use.
		seed time.Time

		// urls contains the URLs to use.
		urls []string

		// expect contains the expectations.
		expect []expectation
	}

	// cases contains test cases.
	cases := []testcase{{
		name:    "without any URL",
		timeNow: nil,
		seed:    time.Time{},
		urls:    []string{},
		expect: []expectation{{
			URL:  "",
			Good: false,
		}, {
			URL:  "",
			Good: false,
		}, {
			URL:  "",
			Good: false,
		}},
	}, {
		name: "with a single URL we get it and then need to wait",
		timeNow: (&testTimeProvider{
			t0: time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC),
			times: []time.Duration{
				0,               // should return URL
				1 * time.Second, // too early to get another URL
				5 * time.Second, // ditto
			},
			idx: 0,
		}).timeNow,
		seed: time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC),
		urls: []string{
			"https://dns.google/dns-query",
		},
		expect: []expectation{{
			URL:  "https://dns.google/dns-query",
			Good: true,
		}, {
			URL:  "",
			Good: false,
		}, {
			URL:  "",
			Good: false,
		}},
	}, {
		name: "with multiple URLs and long wait times we have shuffling",
		timeNow: (&testTimeProvider{
			t0: time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC),
			times: []time.Duration{
				0,                 // should return URL
				60 * time.Minute,  // ditto
				120 * time.Minute, // ditto
			},
			idx: 0,
		}).timeNow,
		seed: time.Date(2024, 2, 8, 9, 8, 7, 6, time.UTC),
		urls: []string{
			"https://dns.google/dns-query",
			"https://cloudflare-dns.com/dns-query",
		},
		expect: []expectation{{
			URL:  "https://cloudflare-dns.com/dns-query",
			Good: true,
		}, {
			URL:  "https://dns.google/dns-query",
			Good: true,
		}, {
			URL:  "https://dns.google/dns-query",
			Good: true,
		}},
	}}

	// run test cases
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oup := NewOpportunisticDNSOverHTTPSURLProvider(tc.urls...)

			// note: we need to reconfigure timeNow before resetting the seed
			if tc.timeNow != nil {
				oup.timeNow = tc.timeNow
			}
			if !tc.seed.IsZero() {
				oup.seed(tc.seed)
			} else {
				oup.seed(oup.timeNow())
			}

			var got []expectation
			for len(got) < len(tc.expect) {
				url, good := oup.MaybeNextURL()
				got = append(got, expectation{
					URL:  url,
					Good: good,
				})
			}

			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
