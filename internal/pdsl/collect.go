package pdsl

import "sync"

// Collect drains a list of channels until they are all closed and returns all the results.
func Collect[T any](chs ...<-chan T) (out []T) {
	wg := &sync.WaitGroup{}
	mtx := &sync.Mutex{}
	for _, ch := range chs {
		wg.Add(1)
		go func(ch <-chan T) {
			defer wg.Done()
			for entry := range ch {
				mtx.Lock()
				out = append(out, entry)
				mtx.Unlock()
			}
		}(ch)
	}
	wg.Wait()
	return
}
