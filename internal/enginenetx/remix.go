package enginenetx

import "sync"

// remix remixes the tactics emitted on Left and Right.
type remix struct {
	// Left is the left channel from which we read the first ReadFromLeft tactics.
	Left <-chan *httpsDialerTactic

	// ReadFromLeft is the number of entries to read from Left at the beginning.
	ReadFromLeft int

	// Right is the right channel from which we read the first ReadFromRight tactics
	// once we've read ReadFromLeft tactics from the Left channel.
	Right <-chan *httpsDialerTactic

	// ReadFromRight is the number of tactics to read from Right once we
	// have read ReadFromLeft tactics from the Left channel.
	ReadFromRight int
}

// Run remixes the Left and Right channel according to its configuration.
//
// The returned channel is closed when both Left and Right are closed.
func (rx *remix) Run() <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		// close the output channel when done
		defer close(output)

		// emit the first N tactics from the left channel
		remixEmitN(rx.Left, rx.ReadFromLeft, output)

		// emit the first M tactics from the right channel
		remixEmitN(rx.Right, rx.ReadFromRight, output)

		// remix all remaining entries
		for tx := range remixDrainBoth(rx.Left, rx.Right) {
			output <- tx
		}
	}()
	return output
}

func remixEmitN(input <-chan *httpsDialerTactic, numToRead int, output chan<- *httpsDialerTactic) {
	for idx := 0; idx < numToRead; idx++ {
		tactic, good := <-input
		if !good {
			return
		}
		output <- tactic
	}
}

func remixDrainBoth(left, right <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
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
