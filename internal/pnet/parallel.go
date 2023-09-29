package pnet

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Parallel runs a given stage in parallel provided that its input type is [Sharable].
//
// The count argument is the number of parallel workers to create. Note that this function
// calls PANIC if the count argument is less than one.
func Parallel[A Sharable, B any](count int, stage Stage[A, B]) Stage[A, B] {
	runtimex.Assert(count >= 1, "the count argument MUST NOT be less than one")
	return StageFunc[A, B](func(ctx context.Context, inputs <-chan Result[A], outputs chan<- Result[B]) {
		// create wait group to track the number of goroutines
		wg := &sync.WaitGroup{}

		for idx := 0; idx < count; idx++ {
			// track that we're creating a goroutinge
			wg.Add(1)

			go func() {
				// notify the wait when done
				defer wg.Done()

				// run the stage in the background with its own output
				intermediate := make(chan Result[B])
				go stage.Run(ctx, inputs, intermediate)

				// drain the intermediate stage's output
				for value := range intermediate {
					outputs <- value
				}
			}()
		}

		// close the outputs channel when done
		defer close(outputs)
		wg.Wait()
	})
}
