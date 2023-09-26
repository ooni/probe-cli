package enginenetx

//
// Scheduling policy based on stats that fallbacks to
// another policy after it has produced all the working
// tactics we can produce given the current stats.
//

import (
	"context"
	"sort"
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
		index := 0
		defer close(out)

		// make sure we don't emit two equal policy in a single run
		uniq := make(map[string]int)

		// function that emits a given tactic unless we already emitted it
		maybeEmitTactic := func(t *httpsDialerTactic) {
			key := t.tacticSummaryKey()
			if uniq[key] > 0 {
				return
			}
			uniq[key]++
			t.InitialDelay = happyEyeballsDelay(index)
			index += 1
			out <- t
		}

		// give priority to what we know from stats
		for _, t := range p.statsLookupTactics(domain, port) {
			maybeEmitTactic(t)
		}

		// fallback to the secondary policy
		for t := range p.Fallback.LookupTactics(ctx, domain, port) {
			maybeEmitTactic(t)
		}
	}()

	return out
}

func (p *statsPolicy) statsLookupTactics(domain string, port string) (out []*httpsDialerTactic) {
	tactics := p.Stats.LookupTactics(domain, port)

	successRate := func(t *statsTactic) (rate float64) {
		if t.CountStarted > 0 {
			rate = float64(t.CountSuccess) / float64(t.CountStarted)
		}
		return
	}

	sort.SliceStable(tactics, func(i, j int) bool {
		// Implementation note: the function should implement the "less" semantics
		// but we want descending sort, so we're using a "more" semantics
		//
		// TODO(bassosimone): should we also consider the number of samples
		// we have and how recent a sample is?
		return successRate(tactics[i]) > successRate(tactics[j])
	})

	for _, t := range tactics {
		// make sure we only include samples with 1+ successes; we don't want this policy
		// to return what we already know it's not working and it will be the purpose of the
		// fallback policy to generate new tactics to test
		if t.CountSuccess > 0 {
			out = append(out, t.Tactic)
		}
	}
	return
}
