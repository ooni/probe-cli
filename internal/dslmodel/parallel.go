package dslmodel

import (
	"context"
	"sync"
)

// Parallel creates a pipeline that runs several copies of the given
// pipeline in parallel. When the N argument is less than one, this func
// just returns the original pipeline to the caller.
func Parallel[A, B any](N int, pipeline Pipeline[A, B]) Pipeline[A, B] {
	// handle the base case
	if N <= 1 {
		return pipeline
	}

	// handle the parallel case
	return PipelineFunc[A, B](func(ctx context.Context, rt Runtime, inputs <-chan Result[A]) <-chan Result[B] {
		// create the overall outputs channel
		outputs := make(chan Result[B])

		// create N goroutines each running the pipeline
		wgroup := &sync.WaitGroup{}
		for idx := 0; idx < N; idx++ {
			wgroup.Add(1)
			go func() {
				defer wgroup.Done()
				for x := range pipeline.Run(ctx, rt, inputs) {
					outputs <- x
				}
			}()
		}

		// create goroutine closing outputs
		go func() {
			defer close(outputs)
			wgroup.Wait()
		}()

		return outputs
	})
}
