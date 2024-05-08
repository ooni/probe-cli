package enginenetx

import (
	"context"
	"slices"
)

var testHelpersDomains = []string{
	"0.th.ooni.org",
	"1.th.ooni.org",
	"2.th.ooni.org",
	"3.th.ooni.org",
	"d33d1gs9kpq1c5.cloudfront.net",
}

// testHelpersPolicy is a policy where we use attempt to
// hide the test helpers domains.
//
// The zero value is invalid; please, init MANDATORY fields.
type testHelpersPolicy struct {
	// Child is the MANDATORY child policy.
	Child httpsDialerPolicy
}

var _ httpsDialerPolicy = &testHelpersPolicy{}

// LookupTactics implements httpsDialerPolicy.
func (p *testHelpersPolicy) LookupTactics(ctx context.Context, domain, port string) <-chan *httpsDialerTactic {
	out := make(chan *httpsDialerTactic)

	go func() {
		defer close(out) // tell the parent when we're done

		for tactic := range p.Child.LookupTactics(ctx, domain, port) {
			// always emit the original tactic first
			out <- tactic

			// When we're not connecting to a TH, our job is done
			if !slices.Contains(testHelpersDomains, tactic.VerifyHostname) {
				continue
			}

			// This is the case where we're connecting to a test helper. Let's try
			// to produce policies using different SNIs for the domain.
			for _, sni := range bridgesDomainsInRandomOrder() {
				out <- &httpsDialerTactic{
					Address:        tactic.Address,
					InitialDelay:   0, // set when dialing
					Port:           tactic.Port,
					SNI:            sni,
					VerifyHostname: tactic.VerifyHostname,
				}
			}
		}
	}()

	return out
}
