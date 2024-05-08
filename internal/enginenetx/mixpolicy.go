package enginenetx

//
// Mix policies - ability of mixing from a primary policy and a fallback policy
// in a more flexible way than strictly falling back
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/optional"
)

// mixPolicyInterleave interleaves policies by a given interleaving
// factor. Say the interleave factor is N, then we first read N tactics
// from the primary policy, then N from the fallback one, and we keep
// going on like this until we've read all the tactics from both.
type mixPolicyInterleave struct {
	// Primary is the primary policy. We will read N from this
	// policy first, then N from fallback, and so on.
	Primary httpsDialerPolicy

	// Fallback is the fallback policy.
	Fallback httpsDialerPolicy

	// Factor is the interleaving factor to use.
	Factor uint8
}

var _ httpsDialerPolicy = &mixPolicyInterleave{}

// LookupTactics implements httpsDialerPolicy.
func (p *mixPolicyInterleave) LookupTactics(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	// create the output channel
	output := make(chan *httpsDialerTactic)

	go func() {
		// make sure we eventually close the output channel
		defer close(output)

		// obtain the primary channel
		primary := optional.Some(p.Primary.LookupTactics(ctx, domain, port))

		// obtain the fallback channel
		fallback := optional.Some(p.Fallback.LookupTactics(ctx, domain, port))

		// loop until both channels are drained
		for !primary.IsNone() || !fallback.IsNone() {
			// take N from primary if possible
			primary = p.maybeTakeN(primary, output)

			// take N from secondary if possible
			fallback = p.maybeTakeN(fallback, output)
		}
	}()

	return output
}

// maybeTakeN takes N entries from input if it's not none. When input is not
// none and reading from it indicates EOF, this function returns none. Otherwise,
// it returns the same value given as input.
func (p *mixPolicyInterleave) maybeTakeN(
	input optional.Value[<-chan *httpsDialerTactic],
	output chan<- *httpsDialerTactic,
) optional.Value[<-chan *httpsDialerTactic] {
	// make sure we've not already drained this channel
	if !input.IsNone() {

		// obtain the underlying channel
		ch := input.Unwrap()

		// take N entries from the channel
		for idx := uint8(0); idx < p.Factor; idx++ {

			// attempt to get the next tactic
			tactic, good := <-ch

			// handle the case where the channel has been drained
			if !good {
				return optional.None[<-chan *httpsDialerTactic]()
			}

			// emit the tactic
			output <- tactic
		}
	}

	return input
}
