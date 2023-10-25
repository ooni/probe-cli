package dslx

//
// Functional extensions (core)
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Func is a function f: (context.Context, A) -> B.
type Func[A, B any] interface {
	Apply(ctx context.Context, a *Maybe[A]) *Maybe[B]
}

// Operation adapts a golang function to behave like a Func.
type Operation[A, B any] func(ctx context.Context, a A) *Maybe[B]

// Apply implements Func.
func (op Operation[A, B]) Apply(ctx context.Context, a *Maybe[A]) *Maybe[B] {
	if a.Error != nil {
		return &Maybe[B]{
			Error: a.Error,
			State: *new(B), // zero value
		}
	}
	return op(ctx, a.State)
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
	b := h.f.Apply(ctx, a)
	runtimex.Assert(b != nil, "h.f.Apply returned a nil pointer")
	c := h.g.Apply(ctx, b)
	runtimex.Assert(c != nil, "h.g.Apply returned a nil pointer")
	return c
}
