package dslx

//
// Functional extensions (async code)
//

import (
	"context"
	"sync"
)

// Parallelism is the type used to specify parallelism.
type Parallelism int

// Map applies fx to the list of elements produced
// by a generator channel.
//
// Arguments:
//
// - ctx is the context;
//
// - parallelism is the number of goroutines to use (we'll use
// a single goroutine if parallelism is < 1);
//
// - fx is the function to apply;
//
// - inputs receives arguments on which to apply fx. We expect this
// channel to be closed to signal EOF to the background workers.
//
// The return value is the channel generating fx(a)
// for every a in inputs. This channel will also be closed
// to signal EOF to the consumer.
func Map[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	fx Func[A, *Maybe[B]],
	inputs <-chan A,
) <-chan *Maybe[B] {
	// create channel for returning results
	r := make(chan *Maybe[B])

	// spawn worker goroutines
	wg := &sync.WaitGroup{}
	if parallelism < 1 {
		parallelism = 1
	}
	for i := Parallelism(0); i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range inputs {
				r <- fx.Apply(ctx, a)
			}
		}()
	}

	// close result channel when done
	go func() {
		defer close(r)
		wg.Wait()
	}()

	return r
}

// Parallel executes f1...fn in parallel over the same input.
//
// Arguments:
//
// - ctx is the context;
//
// - parallelism is the number of goroutines to use (we'll use
// a single goroutine if parallelism is < 1);
//
// - input is the input;
//
// - fn is the list of functions.
//
// The return value is the list [fx(a)] for every fx in fn.
func Parallel[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	input A,
	fn ...Func[A, *Maybe[B]],
) []*Maybe[B] {
	c := ParallelAsync(ctx, parallelism, input, StreamList(fn...))
	return Collect(c)
}

// ParallelAsync is like Parallel but deals with channels. We assume the
// input channel will be closed to signal EOF. We will close the output
// channel to signal EOF to the consumer.
func ParallelAsync[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	input A,
	funcs <-chan Func[A, *Maybe[B]],
) <-chan *Maybe[B] {
	// create channel for returning results
	r := make(chan *Maybe[B])

	// spawn worker goroutines
	wg := &sync.WaitGroup{}
	if parallelism < 1 {
		parallelism = 1
	}
	for i := Parallelism(0); i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fx := range funcs {
				r <- fx.Apply(ctx, input)
			}
		}()
	}

	// close result channel when done
	go func() {
		defer close(r)
		wg.Wait()
	}()

	return r
}

// ApplyAsync is equivalent to calling Apply but returns a channel.
func ApplyAsync[A, B any](
	ctx context.Context,
	fx Func[A, *Maybe[B]],
	input A,
) <-chan *Maybe[B] {
	return Map(ctx, Parallelism(1), fx, StreamList(input))
}
