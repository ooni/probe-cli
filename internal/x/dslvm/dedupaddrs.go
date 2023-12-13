package dslvm

import (
	"context"
	"sync"
)

// DedupAddrsStage is a [Stage] that deduplicates IP addresses.
type DedupAddrsStage struct {
	// Inputs contains the MANDATORY channels from which to read IP addresses. We
	// assume that these channels will be closed when done.
	Inputs []<-chan string

	// Output is the MANDATORY channel where we emit the deduplicated IP addresss. We
	// close this channel when all the Inputs have been closed.
	Output chan<- string
}

var _ Stage = &DedupAddrsStage{}

// Run reads possibly duplicate IP addresses from Inputs and emits deduplicated
// IP addresses on Outputs. We close Outputs when done.
func (sx *DedupAddrsStage) Run(ctx context.Context, rtx Runtime) {
	// create a locked map
	var (
		dups = make(map[string]bool)
		mu   = &sync.Mutex{}
	)

	// stream the input channels to the workers
	inputs := make(chan (<-chan string))
	go func() {
		defer close(inputs)
		for _, input := range sx.Inputs {
			inputs <- input
		}
	}()

	// make sure we cap the number of workers we spawn
	const maxworkers = 6
	workers := len(sx.Inputs)
	if workers > maxworkers {
		workers = maxworkers
	}
	waitGroup := &sync.WaitGroup{}
	for idx := 0; idx < workers; idx++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			// get channel to drain
			for input := range inputs {

				// deduplicate
				for address := range input {

					mu.Lock()
					already := dups[address]
					dups[address] = true
					mu.Unlock()

					if already {
						continue
					}

					sx.Output <- address
				}
			}
		}()
	}

	// make sure we close outputs
	defer close(sx.Output)

	// wait for all inputs to be drained
	waitGroup.Wait()
}
