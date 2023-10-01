package pdsl

import "sync"

// Merge reads from multiple channels until they are all closed.
func Merge[T any](vinputs ...<-chan T) <-chan T {
	// create channel for collecting outputs
	outputs := make(chan T)

	// spawn a goroutine per pipeline
	wg := &sync.WaitGroup{}
	for _, inputs := range vinputs {
		wg.Add(1)
		go func(inputs <-chan T) {
			defer wg.Done()
			for input := range inputs {
				outputs <- input
			}
		}(inputs)
	}

	// close the outputs channel when done
	go func() {
		defer close(outputs)
		wg.Wait()
	}()

	return outputs
}
