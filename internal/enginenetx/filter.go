package enginenetx

// filterOutNilTactics filters out nil tactics.
//
// This function returns a channel where we emit the edited
// tactics, and which we clone when we're done.
func filterOutNilTactics(input <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		defer close(output)
		for tx := range input {
			if tx != nil {
				output <- tx
			}
		}
	}()
	return output
}

// filterOnlyKeepUniqueTactics only keeps unique tactics.
//
// This function returns a channel where we emit the edited
// tactics, and which we clone when we're done.
func filterOnlyKeepUniqueTactics(input <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {

		// make sure we close output chan
		defer close(output)

		// useful to make sure we don't emit two equal policy in a single run
		uniq := make(map[string]int)

		for tx := range input {
			// handle the case in which we already emitted a tactic
			key := tx.tacticSummaryKey()
			if uniq[key] > 0 {
				continue
			}
			uniq[key]++

			// emit the tactic
			output <- tx
		}

	}()
	return output
}

// filterAssignInitialDelays assigns initial delays to tactics.
//
// This function returns a channel where we emit the edited
// tactics, and which we clone when we're done.
func filterAssignInitialDelays(input <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {

		// make sure we close output chan
		defer close(output)

		index := 0
		for tx := range input {
			// TODO(bassosimone): what do we do now about the user configured
			// initial delays? Should we declare them as deprecated?

			// rewrite the delays
			tx.InitialDelay = happyEyeballsDelay(index)
			index++

			// emit the tactic
			output <- tx
		}

	}()
	return output
}
