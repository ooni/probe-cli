package dslvm

import "context"

// TakeNStage is a [Stage] that allows N elements with type T to pass and drops subsequent elements.
type TakeNStage[T any] struct {
	// Input contains the MANDATORY channel from which to read T. We
	// assume that this channel will be closed when done.
	Input <-chan T

	// N is the maximum number of entries to allow to pass. Any value
	// lower than zero is equivalent to setting this field to zero.
	N int64

	// Output is the MANDATORY channel emitting [T]. We will close this
	// channel when the Input channel has been closed.
	Output chan<- T
}

// Run runs the stage until completion.
func (sx *TakeNStage[T]) Run(ctx context.Context, rtx Runtime) {
	// make sure we close the output channel
	defer close(sx.Output)

	var count int64
	for element := range sx.Input {

		// if we've already observed N elements, just drop the N+1-th
		if count >= sx.N {
			drop[T](rtx, element)
			continue
		}

		// otherwise increment counter and forward
		count++
		sx.Output <- element
	}
}
