package dslvm

import (
	"context"
	"sync"
)

// TeeAddrsStage is a [Stage] that duplicates the addresses read in Input into
// each of the channels belonging to [Outputs].
type TeeAddrsStage struct {
	// Input is the MANDATORY channel from which we read addresses. We assume
	// this channel is closed when done.
	Input <-chan string

	// Outputs is the MANDATORY list of channels where to duplicate the addresses read
	// from the Input channel. We close all Outputs when done.
	Outputs []chan<- string
}

var _ Stage = &TeeAddrsStage{}

// Run duplicates addresses read in Input into all the given Outputs.
func (sx *TeeAddrsStage) Run(ctx context.Context, rtx Runtime) {
	// make sure we limit the maximum number of goroutines we will create here
	sema := NewSemaphore("teeAddrs", 12)

	waitGroup := &sync.WaitGroup{}
	for addr := range sx.Input {
		for _, output := range sx.Outputs {
			// make sure we can create a new goroutine
			sema.Wait()

			// register that there is a running goroutine
			waitGroup.Add(1)

			go func(addr string, output chan<- string) {
				// make sure we track that this goroutine is done
				defer waitGroup.Done()

				// make sure a new goroutine can start
				defer sema.Signal()

				// duplicate the address in output
				output <- addr
			}(addr, output)
		}
	}

	// make sure all goroutines finished running
	waitGroup.Wait()

	// close all the output channels
	for _, output := range sx.Outputs {
		close(output)
	}
}
