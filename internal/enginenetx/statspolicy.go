package enginenetx

//
// Scheduling policy based on stats that fallbacks to
// another policy after it has produced all the working
// tactics we can produce given the current stats.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// statsPolicy is a policy that schedules tactics already known
// to work based on statistics and defers to a fallback policy
// once it has generated all the tactics known to work.
//
// The zero value of this struct is invalid; please, make sure you
// fill all the fields marked as MANDATORY.
type statsPolicy struct {
	// Fallback is the MANDATORY fallback policy.
	Fallback httpsDialerPolicy

	// Stats is the MANDATORY stats manager.
	Stats *statsManager
}

var _ httpsDialerPolicy = &statsPolicy{}

// LookupTactics implements HTTPSDialerPolicy.
func (p *statsPolicy) LookupTactics(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	out := make(chan *httpsDialerTactic)

	go func() {
		defer close(out) // make sure the parent knows when we're done
		index := 0

		// useful to make sure we don't emit two equal policy in a single run
		uniq := make(map[string]int)

		// function that emits a given tactic unless we already emitted it
		maybeEmitTactic := func(t *httpsDialerTactic) {
			// as a safety mechanism let's gracefully handle the
			// case in which the tactic is nil
			if t == nil {
				return
			}

			// handle the case in which we already emitted a policy
			key := t.tacticSummaryKey()
			if uniq[key] > 0 {
				return
			}
			uniq[key]++

			// 🚀!!!
			t.InitialDelay = 0 // set when dialing
			index += 1
			out <- t
		}

		// give priority to what we know from stats
		for _, t := range statsPolicyFilterStatsTactics(p.Stats.LookupTactics(domain, port)) {
			maybeEmitTactic(t)
		}

		// fallback to the secondary policy
		for t := range p.Fallback.LookupTactics(ctx, domain, port) {
			maybeEmitTactic(t)
		}
	}()

	return out
}

func statsPolicyFilterStatsTactics(tactics []*statsTactic, good bool) (out []*httpsDialerTactic) {
	// when good is false, it means p.Stats.LookupTactics failed
	if !good {
		return
	}

	// only keep well-formed successful entries
	onlySuccesses := statsDefensivelySortTacticsByDescendingSuccessRateWithAcceptPredicate(
		tactics, func(st *statsTactic) bool {
			return st != nil && st.Tactic != nil && st.CountSuccess > 0
		},
	)

	// convert the statsTactic list into a list of tactics
	for _, t := range onlySuccesses {
		runtimex.Assert(t != nil && t.Tactic != nil && t.CountSuccess > 0, "expected well-formed *statsTactic")
		out = append(out, t.Tactic)
	}
	return
}
