package dslmodel

import (
	"context"
	"sync"
)

// Distribute distributes [Sharable] work across several pipelines running
// in parallel and fans-in the results into a single channel.
func Distribute[A Sharable, B any](pxs ...Pipeline[A, B]) Pipeline[A, B] {
	return PipelineFunc[A, B](func(ctx context.Context, rt Runtime, inputs <-chan Result[A]) <-chan Result[B] {
		// create an input channel for each goroutine
		var vinputs []chan Result[A]
		for idx := 0; idx < len(pxs); idx++ {
			vinputs = append(vinputs, make(chan Result[A]))
		}

		// create channel for collecting outputs
		outputs := make(chan Result[B])

		// spawn a goroutine per pipeline
		wgroup := &sync.WaitGroup{}
		for idx := 0; idx < len(pxs); idx++ {
			wgroup.Add(1)
			go func(idx int) {
				defer wgroup.Done()
				for e := range pxs[idx].Run(ctx, rt, vinputs[idx]) {
					outputs <- e
				}
			}(idx)
		}

		// close the outputs channel when done
		go func() {
			defer close(outputs)
			wgroup.Wait()
		}()

		// read and distribute inputs then close pipelines
		go func() {
			for e := range inputs {
				for _, ch := range vinputs {
					ch <- e
				}
			}
			for _, ch := range vinputs {
				close(ch)
			}
		}()

		return outputs
	})
}
