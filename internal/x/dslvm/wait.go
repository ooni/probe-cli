package dslvm

import "sync"

// Wait waits until all the given channels are done.
func Wait(channels ...<-chan Done) {
	waitGroup := &sync.WaitGroup{}
	for _, channel := range channels {
		waitGroup.Add(1)
		go func(channel <-chan Done) {
			defer waitGroup.Done()
			for range channel {
				// drain!
			}
		}(channel)
	}
	waitGroup.Wait()
}
