package pdsl

import "sync"

// Drain drains a list of channels until they are all closed.
func Drain[T any](chs ...<-chan T) {
	wg := &sync.WaitGroup{}
	for _, ch := range chs {
		wg.Add(1)
		go func(ch <-chan T) {
			defer wg.Done()
			for range ch {
				// nothing!
			}
		}(ch)
	}
	wg.Wait()
}
