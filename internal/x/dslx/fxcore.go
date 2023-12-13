package dslx

//
// Functional extensions (core)
//

import (
	"context"
	"errors"
	"sync"
)

// Func is a function f: (context.Context, A) -> B.
type Func[A, B any] interface {
	Apply(ctx context.Context, a *Maybe[A]) *Maybe[B]
}

// FuncAdapter adapts a func to be a [Func].
type FuncAdapter[A, B any] func(ctx context.Context, a *Maybe[A]) *Maybe[B]

// Apply implements Func.
func (fa FuncAdapter[A, B]) Apply(ctx context.Context, a *Maybe[A]) *Maybe[B] {
	return fa(ctx, a)
}

// Operation adapts a golang function to behave like a Func.
type Operation[A, B any] func(ctx context.Context, a A) (B, error)

// Apply implements Func.
func (op Operation[A, B]) Apply(ctx context.Context, a *Maybe[A]) *Maybe[B] {
	if err := a.Error; err != nil {
		return NewMaybeWithError[B](err)
	}
	out, err := op(ctx, a.State)
	if err != nil {
		return NewMaybeWithError[B](err)
	}
	return NewMaybeWithValue(out)
}

// Maybe is the result of an operation implemented by this package
// that may fail such as [TCPConnect] or [TLSHandshake].
type Maybe[State any] struct {
	// Error is either the error that occurred or nil.
	Error error

	// State contains state passed between function calls. You should
	// only access State when Error is nil and Skipped is false.
	State State
}

// NewMaybeWithValue constructs a Maybe containing the given value.
func NewMaybeWithValue[State any](value State) *Maybe[State] {
	return &Maybe[State]{
		Error: nil,
		State: value,
	}
}

// NewMaybeWithError constructs a Maybe containing the given error.
func NewMaybeWithError[State any](err error) *Maybe[State] {
	return &Maybe[State]{
		Error: err,
		State: *new(State), // zero value
	}
}

// Compose2 composes two operations such as [TCPConnect] and [TLSHandshake].
func Compose2[A, B, C any](f Func[A, B], g Func[B, C]) Func[A, C] {
	return &compose2Func[A, B, C]{
		f: f,
		g: g,
	}
}

// compose2Func is the type returned by [Compose2].
type compose2Func[A, B, C any] struct {
	f Func[A, B]
	g Func[B, C]
}

// Apply implements Func
func (h *compose2Func[A, B, C]) Apply(ctx context.Context, a *Maybe[A]) *Maybe[C] {
	return h.g.Apply(ctx, h.f.Apply(ctx, a))
}

// Void is the empty data structure.
type Void struct{}

// Discard transforms any type to [Void].
func Discard[T any]() Func[T, Void] {
	return Operation[T, Void](func(ctx context.Context, input T) (Void, error) {
		return Void{}, nil
	})
}

// ErrSkip is an error that indicates that we already processed an error emitted
// by a previous stage, so we are using this error to avoid counting the original
// error more than once when computing statistics, e.g., in [*Stats].
var ErrSkip = errors.New("dslx: error already processed by a previous stage")

// Stats measures the number of successes and failures.
//
// The zero value is invalid; use [NewStats].
type Stats[T any] struct {
	m  map[string]int64
	mu sync.Mutex
}

// NewStats creates a [*Stats] instance.
func NewStats[T any]() *Stats[T] {
	return &Stats[T]{
		m:  map[string]int64{},
		mu: sync.Mutex{},
	}
}

// Observer returns a Func that observes the results of the previous pipeline stage. This function
// converts any error that it sees to [ErrSkip]. This function does not account for [ErrSkip], meaning
// that you will never see [ErrSkip] in the stats returned by [Stats.Export].
func (s *Stats[T]) Observer() Func[T, T] {
	return FuncAdapter[T, T](func(ctx context.Context, minput *Maybe[T]) *Maybe[T] {
		defer s.mu.Unlock()
		s.mu.Lock()
		var r string
		if err := minput.Error; err != nil {
			if errors.Is(err, ErrSkip) {
				return NewMaybeWithError[T](ErrSkip) // as documented
			}
			r = err.Error()
		}
		s.m[r]++
		if r != "" {
			return NewMaybeWithError[T](ErrSkip) // as documented
		}
		return minput
	})
}

// Export exports the current stats without clearing the internally used map such that
// statistics accumulate over time and never reset for the [*Stats] lifecycle.
func (s *Stats[T]) Export() (out map[string]int64) {
	out = make(map[string]int64)
	defer s.mu.Unlock()
	s.mu.Lock()
	for r, cnt := range s.m {
		out[r] = cnt
	}
	return
}
