package enginenetx

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMixPolicyEitherOr(t *testing.T) {
	// testcase is a test case implemented by this function.
	type testcase struct {
		// name is the name of the test case
		name string

		// primary is the primary policy to use
		primary httpsDialerPolicy

		// fallback is the fallback policy to use
		fallback httpsDialerPolicy

		// domain is the domain to pass to LookupTactics
		domain string

		// port is the port to pass to LookupTactics
		port string

		// expect is the expectations in terms of tactics
		expect []*httpsDialerTactic
	}

	// This is the list of tactics that we expect the primary
	// policy to return when we're not using a null policy
	expectedPrimaryTactics := []*httpsDialerTactic{{
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "shelob.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "whitespider.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "mirkwood.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "highgarden.polito.it",
		VerifyHostname: "api.ooni.io",
	}}

	// Create the non-null primary policy
	primary := &userPolicyV2{
		Root: &userPolicyRoot{
			DomainEndpoints: map[string][]*httpsDialerTactic{
				"api.ooni.io:443": expectedPrimaryTactics,
			},
			Version: userPolicyVersion,
		},
	}

	// This is the list of tactics that we expect the fallback
	// policy to return when we're not using a null policy
	expectedFallbackTactics := []*httpsDialerTactic{{
		Address:        "130.192.91.231",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "kingslanding.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.231",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "pyke.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.231",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "winterfell.polito.it",
		VerifyHostname: "api.ooni.io",
	}}

	// Create the non-null fallback policy
	fallback := &userPolicyV2{
		Root: &userPolicyRoot{
			DomainEndpoints: map[string][]*httpsDialerTactic{
				"api.ooni.io:443": expectedFallbackTactics,
			},
			Version: userPolicyVersion,
		},
	}

	cases := []testcase{

		// This test ensures that the code is WAI with two null policies
		{
			name:     "with two null policies",
			primary:  &nullPolicy{},
			fallback: &nullPolicy{},
			domain:   "api.ooni.io",
			port:     "443",
			expect:   nil,
		},

		// This test ensures that we get the content of the primary
		// policy when the fallback policy is the null policy
		{
			name:     "with the fallback policy being null",
			primary:  primary,
			fallback: &nullPolicy{},
			domain:   "api.ooni.io",
			port:     "443",
			expect:   expectedPrimaryTactics,
		},

		// This test ensures that we get the content of the fallback
		// policy when the primary policy is the null policy
		{
			name:     "with the primary policy being null",
			primary:  &nullPolicy{},
			fallback: fallback,
			domain:   "api.ooni.io",
			port:     "443",
			expect:   expectedFallbackTactics,
		},

		// This test ensures that we correctly only get the primary
		// policy when both primary and fallback are set
		{
			name:     "with both policies being nonnull",
			primary:  primary,
			fallback: fallback,
			domain:   "api.ooni.io",
			port:     "443",
			expect:   expectedPrimaryTactics,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// construct the mixPolicyEitherOr instance
			p := &mixPolicyEitherOr{
				Primary:  tc.primary,
				Fallback: tc.fallback,
			}

			// start looking up for tactics
			outch := p.LookupTactics(context.Background(), tc.domain, tc.port)

			// collect all the generated tactics
			var got []*httpsDialerTactic
			for entry := range outch {
				got = append(got, entry)
			}

			// compare to expectations
			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestMixPolicyInterleave(t *testing.T) {
	// testcase is a test case implemented by this function.
	type testcase struct {
		// name is the name of the test case
		name string

		// primary is the primary policy to use
		primary httpsDialerPolicy

		// fallback is the fallback policy to use
		fallback httpsDialerPolicy

		// factor is the interleave factor
		factor uint8

		// domain is the domain to pass to LookupTactics
		domain string

		// port is the port to pass to LookupTactics
		port string

		// expect is the expectations in terms of tactics
		expect []*httpsDialerTactic
	}

	// This is the list of tactics that we expect the primary
	// policy to return when we're not using a null policy
	expectedPrimaryTactics := []*httpsDialerTactic{{
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "shelob.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "whitespider.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "mirkwood.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "highgarden.polito.it",
		VerifyHostname: "api.ooni.io",
	}}

	// Create the non-null primary policy
	primary := &userPolicyV2{
		Root: &userPolicyRoot{
			DomainEndpoints: map[string][]*httpsDialerTactic{
				"api.ooni.io:443": expectedPrimaryTactics,
			},
			Version: userPolicyVersion,
		},
	}

	// This is the list of tactics that we expect the fallback
	// policy to return when we're not using a null policy
	expectedFallbackTactics := []*httpsDialerTactic{{
		Address:        "130.192.91.231",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "kingslanding.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.231",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "pyke.polito.it",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.231",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "winterfell.polito.it",
		VerifyHostname: "api.ooni.io",
	}}

	// Create the non-null fallback policy
	fallback := &userPolicyV2{
		Root: &userPolicyRoot{
			DomainEndpoints: map[string][]*httpsDialerTactic{
				"api.ooni.io:443": expectedFallbackTactics,
			},
			Version: userPolicyVersion,
		},
	}

	cases := []testcase{

		// This test ensures that the code is WAI with two null policies
		{
			name:     "with two null policies",
			primary:  &nullPolicy{},
			fallback: &nullPolicy{},
			factor:   0,
			domain:   "api.ooni.io",
			port:     "443",
			expect:   nil,
		},

		// This test ensures that we get the content of the primary
		// policy when the fallback policy is the null policy
		{
			name:     "with the fallback policy being null",
			primary:  primary,
			fallback: &nullPolicy{},
			factor:   0,
			domain:   "api.ooni.io",
			port:     "443",
			expect:   expectedPrimaryTactics,
		},

		// This test ensures that we get the content of the fallback
		// policy when the primary policy is the null policy
		{
			name:     "with the primary policy being null",
			primary:  &nullPolicy{},
			fallback: fallback,
			factor:   0,
			domain:   "api.ooni.io",
			port:     "443",
			expect:   expectedFallbackTactics,
		},

		// This test ensures that we correctly interleave the tactics
		{
			name:     "with both policies being nonnull and interleave being nonzero",
			primary:  primary,
			fallback: fallback,
			factor:   2,
			domain:   "api.ooni.io",
			port:     "443",
			expect: []*httpsDialerTactic{
				expectedPrimaryTactics[0],
				expectedPrimaryTactics[1],
				expectedFallbackTactics[0],
				expectedFallbackTactics[1],
				expectedPrimaryTactics[2],
				expectedPrimaryTactics[3],
				expectedFallbackTactics[2],
			},
		},

		// This test ensures that we behave correctly when factor is zero
		{
			name:     "with both policies being nonnull and interleave being zero",
			primary:  primary,
			fallback: fallback,
			factor:   0,
			domain:   "api.ooni.io",
			port:     "443",
			expect: []*httpsDialerTactic{
				expectedPrimaryTactics[0],
				expectedFallbackTactics[0],
				expectedPrimaryTactics[1],
				expectedFallbackTactics[1],
				expectedPrimaryTactics[2],
				expectedFallbackTactics[2],
				expectedPrimaryTactics[3],
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// construct the mixPolicyInterleave instance
			p := &mixPolicyInterleave{
				Primary:  tc.primary,
				Fallback: tc.fallback,
				Factor:   tc.factor,
			}

			// start looking up for tactics
			outch := p.LookupTactics(context.Background(), tc.domain, tc.port)

			// collect all the generated tactics
			var got []*httpsDialerTactic
			for entry := range outch {
				got = append(got, entry)
			}

			// compare to expectations
			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
