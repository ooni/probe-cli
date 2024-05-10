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

// statsPolicyV2 is a policy that schedules tactics already known
// to work based on the previously collected stats.
//
// The zero value of this struct is invalid; please, make sure
// you fill all the fields marked as MANDATORY.
//
// This is v2 of the statsPolicy because the previous implementation
// incorporated mixing logic, while now the mixing happens outside
// of this policy, thus giving us much more flexibility.
type statsPolicyV2 struct {
	// Stats is the MANDATORY stats manager.
	Stats *statsManager
}

var _ httpsDialerPolicy = &statsPolicyV2{}

// LookupTactics implements httpsDialerPolicy.
func (p *statsPolicyV2) LookupTactics(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	return streamTacticsFromSlice(statsPolicyFilterStatsTactics(p.Stats.LookupTactics(domain, port)))
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
