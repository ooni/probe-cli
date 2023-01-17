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

// Map applies fx to a list of elements.
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
// - as is the list on which to apply fx.
//
// The return value is the list [fx(a)] for every a in as.
func Map[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	fx Func[A, *Maybe[B]],
	as ...A,
) []*Maybe[B] {
	return MapAsync(ctx, parallelism, fx, Stream(as...)).Collect()
}

// MapAsync is like Map but deals with streams.
func MapAsync[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	fx Func[A, *Maybe[B]],
	inputs *Streamable[A],
) *Streamable[*Maybe[B]] {
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
			for a := range inputs.C {
				r <- fx.Apply(ctx, a)
			}
		}()
	}

	// close channel when done
	go func() {
		defer close(r)
		wg.Wait()
	}()

	return &Streamable[*Maybe[B]]{r}
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
	return ParallelAsync(ctx, parallelism, input, Stream(fn...)).Collect()
}

// ParallelAsync is like Parallel but deals with streams.
func ParallelAsync[A, B any](
	ctx context.Context,
	parallelism Parallelism,
	input A,
	funcs *Streamable[Func[A, *Maybe[B]]],
) *Streamable[*Maybe[B]] {
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
			for fx := range funcs.C {
				r <- fx.Apply(ctx, input)
			}
		}()
	}

	// close channel when done
	go func() {
		defer close(r)
		wg.Wait()
	}()

	return &Streamable[*Maybe[B]]{r}
}

// ApplyAsync is equivalent to calling Apply but returns a Streamable.
func ApplyAsync[A, B any](
	ctx context.Context,
	fx Func[A, *Maybe[B]],
	input A,
) *Streamable[*Maybe[B]] {
	return MapAsync(ctx, Parallelism(1), fx, Stream(input))
}
