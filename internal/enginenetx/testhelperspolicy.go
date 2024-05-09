package enginenetx

import (
	"context"
	"slices"
)

// testHelpersDomains is our understanding of TH domains.
var testHelpersDomains = []string{
	"0.th.ooni.org",
	"1.th.ooni.org",
	"2.th.ooni.org",
	"3.th.ooni.org",
	"d33d1gs9kpq1c5.cloudfront.net",
}

// testHelpersPolicy is a policy where we extend TH related policies
// by adding additional SNIs that it makes sense to try.
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
		// tell the parent when we're done
		defer close(out)

		// collect tactics that we may want to modify later
		var todo []*httpsDialerTactic

		// always emit the original tactic first
		//
		// See https://github.com/ooni/probe-cli/pull/1552 review for
		// a rationale of why we're emitting the original first
		for tactic := range p.Child.LookupTactics(ctx, domain, port) {
			out <- tactic

			// When we're not connecting to a TH, our job is done
			if !slices.Contains(testHelpersDomains, tactic.VerifyHostname) {
				continue
			}

			// otherwise, let's rememeber to modify this later
			todo = append(todo, tactic)
		}

		// This is the case where we're connecting to a test helper. Let's try
		// to produce tactics using different SNIs for the domain.
		for _, tactic := range todo {
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
