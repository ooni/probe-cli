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
	// avoid emitting nil tactics and duplicate tactics
	return filterOnlyKeepUniqueTactics(filterOutNilTactics(mixDeterministicThenRandom(
		&mixDeterministicThenRandomConfig{
			// Give priority to what we know from stats.
			C: streamTacticsFromSlice(statsPolicyFilterStatsTactics(p.Stats.LookupTactics(domain, port))),

			// We make sure we emit two stats-based tactics if possible.
			N: 2,
		},

		&mixDeterministicThenRandomConfig{
			// And remix it with the fallback.
			C: p.Fallback.LookupTactics(ctx, domain, port),

			// Under the assumption that below us we have bridgePolicy composed with DNS policy
			// and that the stage below emits two bridge tactics, if possible, followed by two
			// additional DNS tactics, if possible, we need to allow for four tactics to pass through
			// befofe we start remixing from the two channels.
			//
			// Note: modifying this field likely indicates you also need to modify the
			// corresponding instantiation in bridgespolicy.go.
			N: 4,
		},
	)))
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
