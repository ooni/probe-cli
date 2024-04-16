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
	rx := &remix{
		// Give priority to what we know from stats
		Left: statsPolicyStream(statsPolicyFilterStatsTactics(p.Stats.LookupTactics(domain, port))),

		// We make sure we emit two stats-based tactics if possible
		ReadFromLeft: 2,

		// And remix it with the fallback
		Right: p.onlyAccessibleEndpoints(p.Fallback.LookupTactics(ctx, domain, port)),

		// Under the assumption that below us we have bridgePolicy composed with DNS policy
		// and that the stage below emits two bridge tactics, if possible, followed by two
		// additional DNS tactics, if possible, we need to allow for four tactics to pass through
		// befofe we start remixing from the two channels.
		//
		// Note: modifying this field likely indicates you also need to modify the
		// corresponding remix{} instantiation in bridgespolicy.go.
		ReadFromRight: 4,
	}
	return rx.Run()
}

// statsPolicyStream streams a vector of tactics.
func statsPolicyStream(txs []*httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		defer close(output)
		for _, tx := range txs {
			output <- tx
		}
	}()
	return output
}

// statsPolicyFilterStatsTactics filters the tactics generated by consulting the stats.
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

// onlyAccessibleEndpoints uses stats-based knowledge to exclude using endpoints that
// have recently been observed as being failing during TCP connect.
func (p *statsPolicy) onlyAccessibleEndpoints(input <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		// make sure we close the output channel
		defer close(output)

		// avoid including tactics using endpoints that are consistently failing
		for tx := range input {
			if tx == nil || !p.Stats.IsTCPEndpointAccessible(tx.Address, tx.Port) {
				continue
			}
			output <- tx
		}
	}()
	return output
}
