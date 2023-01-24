package dslx

//
// Functional extensions (stream)
//

import "sync"

// Collect collects all the elements inside a channel.
func Collect[T any](c <-chan T) (v []T) {
	for t := range c { // the producer closes C when done
		v = append(v, t)
	}
	return
}

// StreamList creates a channel out of static values.
func StreamList[T any](ts ...T) <-chan T {
	c := make(chan T)
	go func() {
		defer close(c) // as documented
		for _, t := range ts {
			c <- t
		}
	}()
	return c
}

// Zip zips together results from many [Streabable]s.
func Zip[T any](sources ...<-chan T) <-chan T {
	r := make(chan T)
	wg := &sync.WaitGroup{}
	for _, src := range sources {
		wg.Add(1)
		go func(c <-chan T) {
			defer wg.Done()
			for e := range c { // the producer closes C when done
				r <- e
			}
		}(src)
	}
	go func() {
		defer close(r) // as documented
		wg.Wait()
	}()
	return r
}

// ZipAndCollect is syntactic sugar for Collect(Zip(sources...)).
func ZipAndCollect[T any](sources ...<-chan T) []T {
	return Collect(Zip(sources...))
}
