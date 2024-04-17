package enginenetx

import "sync"

// mixSequentially mixes entries from primary followed by entries from fallback.
//
// This function returns a channel where we emit the edited
// tactics, and which we clone when we're done.
func mixSequentially(primary, fallback <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		defer close(output)
		for tx := range primary {
			output <- tx
		}
		for tx := range fallback {
			output <- tx
		}
	}()
	return output
}

// mixDeterministicThenRandomConfig contains config for [mixDeterministicThenRandom].
type mixDeterministicThenRandomConfig struct {
	// C is the channel to mix from.
	C <-chan *httpsDialerTactic

	// N is the number of entries to read from at the
	// beginning before starting random mixing.
	N int
}

// mixDeterministicThenRandom reads the first N entries from primary, if any, then the first N
// entries from fallback, if any, and then randomly mixes the entries.
func mixDeterministicThenRandom(primary, fallback *mixDeterministicThenRandomConfig) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		defer close(output)
		mixTryEmitN(primary.C, primary.N, output)
		mixTryEmitN(fallback.C, fallback.N, output)
		for tx := range mixRandomly(primary.C, fallback.C) {
			output <- tx
		}
	}()
	return output
}

func mixTryEmitN(input <-chan *httpsDialerTactic, numToRead int, output chan<- *httpsDialerTactic) {
	for idx := 0; idx < numToRead; idx++ {
		tactic, good := <-input
		if !good {
			return
		}
		output <- tactic
	}
}

func mixRandomly(left, right <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		// read from left
		waitg := &sync.WaitGroup{}
		waitg.Add(1)
		go func() {
			defer waitg.Done()
			for tx := range left {
				output <- tx
			}
		}()

		// read from right
		waitg.Add(1)
		go func() {
			defer waitg.Done()
			for tx := range right {
				output <- tx
			}
		}()

		// close when done
		go func() {
			waitg.Wait()
			close(output)
		}()
	}()
	return output
}
