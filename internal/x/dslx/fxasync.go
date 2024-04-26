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
//
// Deprecated: use Matrix instead.
func Map[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	fx Func[A, B],
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
				r <- fx.Apply(ctx, NewMaybeWithValue(a))
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
//
// Deprecated: use Matrix instead.
func Parallel[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	input A,
	fn ...Func[A, B],
) []*Maybe[B] {
	c := ParallelAsync(ctx, parallelism, input, StreamList(fn...))
	return Collect(c)
}

// ParallelAsync is like Parallel but deals with channels. We assume the
// input channel will be closed to signal EOF. We will close the output
// channel to signal EOF to the consumer.
//
// Deprecated: use Matrix instead.
func ParallelAsync[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	input A,
	funcs <-chan Func[A, B],
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
				r <- fx.Apply(ctx, NewMaybeWithValue(input))
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
//
// Deprecated: use Matrix instead.
func ApplyAsync[A, B any](
	ctx context.Context,
	fx Func[A, B],
	input A,
) <-chan *Maybe[B] {
	return Map(ctx, Parallelism(1), fx, StreamList(input))
}

// matrixPoint is a point within the matrix used by [Matrix].
type matrixPoint[A, B any] struct {
	f  Func[A, B]
	in A
}

// Matrix invokes each function on each input using N goroutines and streams the results to a channel.
func Matrix[A, B any](ctx context.Context, N Parallelism, inputs []A, functions []Func[A, B]) <-chan *Maybe[B] {
	// make output
	output := make(chan *Maybe[B])

	// stream all the possible points
	points := make(chan *matrixPoint[A, B])
	go func() {
		defer close(points)
		for _, input := range inputs {
			for _, fx := range functions {
				points <- &matrixPoint[A, B]{f: fx, in: input}
			}
		}
	}()

	// spawn goroutines
	wg := &sync.WaitGroup{}
	N = min(1, N)
	for i := Parallelism(0); i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range points {
				output <- p.f.Apply(ctx, NewMaybeWithValue(p.in))
			}
		}()
	}

	// close output channel when done
	go func() {
		defer close(output)
		wg.Wait()
	}()

	return output
}
