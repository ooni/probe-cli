package dslx

//
// Functional extensions (stream)
//

import "sync"

// Streamable wraps a channel that returns T and is closed
// by the producer when all input has been emitted.
type Streamable[T any] struct {
	// C is the channel written by the producer.
	C <-chan T
}

// Collect collects all the elements inside a stream.
func (s *Streamable[T]) Collect() (v []T) {
	for t := range s.C { // the producer closes C when done
		v = append(v, t)
	}
	return
}

// Stream creates a Streamable out of static values.
func Stream[T any](ts ...T) *Streamable[T] {
	c := make(chan T)
	go func() {
		defer close(c) // as documented
		for _, t := range ts {
			c <- t
		}
	}()
	return &Streamable[T]{c}
}

// Zip zips together results from many [Streabable]s.
func Zip[T any](sources ...*Streamable[T]) *Streamable[T] {
	r := make(chan T)
	wg := &sync.WaitGroup{}
	for _, src := range sources {
		wg.Add(1)
		go func(s *Streamable[T]) {
			defer wg.Done()
			for e := range s.C { // the producer closes C when done
				r <- e
			}
		}(src)
	}
	go func() {
		defer close(r) // as documented
		wg.Wait()
	}()
	return &Streamable[T]{r}
}

// ZipAndCollect is syntactic sugar for Zip(sources...).Collect().
func ZipAndCollect[T any](sources ...*Streamable[T]) []T {
	return Zip(sources...).Collect()
}
