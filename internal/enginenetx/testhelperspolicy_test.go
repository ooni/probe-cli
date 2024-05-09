package enginenetx

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTestHelpersPolicy(t *testing.T) {

	// testHelperTactics contains tactics related to test helpers
	testHelperTactics := []*httpsDialerTactic{{
		Address:        "18.195.190.71",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "0.th.ooni.org",
		VerifyHostname: "0.th.ooni.org",
	}, {
		Address:        "18.198.214.127",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "0.th.ooni.org",
		VerifyHostname: "0.th.ooni.org",
	}}

	// wwwExampleComTactics contains tactics related to www.example.com
	wwwExampleComTactic := []*httpsDialerTactic{{
		Address:        "93.184.215.14",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "www.example.com",
		VerifyHostname: "www.example.com",
	}, {
		Address:        "2606:2800:21f:cb07:6820:80da:af6b:8b2c",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "www.example.com",
		VerifyHostname: "www.example.com",
	}}

	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the test case name
		name string

		// childTactics contains the tactics that the child policy
		// should return when invoked by the policy
		childTactics []*httpsDialerTactic

		// domain is the domain to attempt to obtain tactics for
		domain string

		// expectExtra contains the number of expected tactics
		// we want to see beyond the child tactics above
		expectExtra int
	}

	cases := []testcase{{
		name:         "when the children does not return any tactic, duh",
		childTactics: nil,
		domain:       "www.example.com",
		expectExtra:  0,
	}, {
		name:         "when the children returns a non-TH domain",
		childTactics: wwwExampleComTactic,
		domain:       wwwExampleComTactic[0].VerifyHostname,
		expectExtra:  0,
	}, {
		name:         "when the children returns a TH domain",
		childTactics: testHelperTactics,
		domain:       testHelperTactics[0].VerifyHostname,
		expectExtra:  304,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// create the policy that we're testing
			//
			// note how the child policy is just returning the expected
			// set of child tactics in the original order
			policy := &testHelpersPolicy{
				Child: &mocksPolicy{
					MockLookupTactics: func(ctx context.Context, domain, port string) <-chan *httpsDialerTactic {
						output := make(chan *httpsDialerTactic)
						go func() {
							defer close(output)
							for _, entry := range tc.childTactics {
								output <- entry
							}
						}()
						return output
					},
				},
			}

			// start to generate tactics for the given domain
			generator := policy.LookupTactics(context.Background(), tc.domain, "443")

			// obtain all the tactics
			var tactics []*httpsDialerTactic
			for entry := range generator {
				tactics = append(tactics, entry)
			}

			// make sure we have the expected number of tactics
			// at the beginning of the list
			if len(tactics) < len(tc.childTactics) {
				t.Fatal("expected at least", len(tc.childTactics), "got", len(tactics))
			}

			// if there are expected tactics make sure they
			// indeed match our expectations
			if len(tc.childTactics) > 0 {
				if diff := cmp.Diff(tc.childTactics, tactics[:len(tc.childTactics)]); diff != "" {
					t.Fatal(diff)
				}
			}

			// make sure we have the expected nymber of extras
			if diff := len(tactics) - len(tc.childTactics); diff != tc.expectExtra {
				t.Fatal("expected", tc.expectExtra, "extras but got", diff)
				return
			}

			// if the expected number of extras is zero, what are we still
			// doing here and why don't we return like now?
			if tc.expectExtra <= 0 {
				return
			}

			// make sure we're not going to expose the domain via the SNI
			for _, entry := range tactics[len(tc.childTactics):] {
				if entry.SNI == tc.domain {
					t.Fatal("did not expect to see", tc.domain, "but got", entry.SNI)
				}
				if entry.VerifyHostname != tc.domain {
					t.Fatal("expected to see", tc.domain, "but got", entry.VerifyHostname)
				}
			}
		})
	}
}
