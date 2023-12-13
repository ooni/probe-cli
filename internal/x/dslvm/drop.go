package dslvm

import "context"

// DropStage is a [Stage] that drops reference to whatever it is passed in input. If the
// input is a [Closer], this stage will also make sure it is closed.
type DropStage[T any] struct {
	// Input contains the MANDATORY channel from which to read instances to drop. We
	// assume that this channel will be closed when done.
	Input <-chan T

	// Output contains the MANDATORY channel closed when Input has been closed.
	Output chan Done
}

var _ Stage = &DropStage[*TCPConnection]{}

// Run drops all the input passed to the Input channel and closes Output when done.
func (sx *DropStage[T]) Run(ctx context.Context, rtx Runtime) {
	// make sure we close Output when done
	defer close(sx.Output)

	for input := range sx.Input {
		drop[T](rtx, input)
	}
}

func drop[T any](rtx Runtime, value any) {
	if closer, good := any(value).(Closer); good {
		// close the connection and log about it
		_ = closer.Close(rtx.Logger())

		// make sure we signal the semaphore
		rtx.ActiveConnections().Signal()
	}
}
