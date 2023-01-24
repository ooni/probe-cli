package dslx

//
// Functional extensions (core)
//

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Func is a function f: (context.Context, A) -> B.
type Func[A, B any] interface {
	Apply(ctx context.Context, a A) B
}

// Maybe is the result of an operation implemented by this package
// that may fail such as [TCPConnect] or [TLSHandshake].
type Maybe[State any] struct {
	// Error is either the error that occurred or nil.
	Error error

	// Observations contains the collected observations.
	Observations []*Observations

	// Skipped indicates whether an operation decided
	// that subsequent steps should be skipped.
	Skipped bool

	// State contains state passed between function calls. You should
	// only access State when Error is nil and Skipped is false.
	State State
}

// Compose2 composes two operations such as [TCPConnect] and [TLSHandshake].
func Compose2[A, B, C any](f Func[A, *Maybe[B]], g Func[B, *Maybe[C]]) Func[A, *Maybe[C]] {
	return &compose2Func[A, B, C]{
		f: f,
		g: g,
	}
}

// compose2Func is the type returned by [Compose2].
type compose2Func[A, B, C any] struct {
	f Func[A, *Maybe[B]]
	g Func[B, *Maybe[C]]
}

// Apply implements Func
func (h *compose2Func[A, B, C]) Apply(ctx context.Context, a A) *Maybe[C] {
	mb := h.f.Apply(ctx, a)
	runtimex.Assert(mb != nil, "h.f.Apply returned a nil pointer")
	if mb.Skipped || mb.Error != nil {
		return &Maybe[C]{
			Error:        mb.Error,
			Observations: mb.Observations,
			Skipped:      mb.Skipped,
			State:        *new(C), // zero value
		}
	}
	mc := h.g.Apply(ctx, mb.State)
	runtimex.Assert(mc != nil, "h.g.Apply returned a nil pointer")
	return &Maybe[C]{
		Error:        mc.Error,
		Observations: append(mb.Observations, mc.Observations...), // merge observations
		Skipped:      mc.Skipped,
		State:        mc.State,
	}
}

// NewCounter generates an instance of *Counter
func NewCounter[T any]() *Counter[T] {
	return &Counter[T]{}
}

// Counter allows to count how many times
// a Func[T, *Maybe[T]] is invoked.
type Counter[T any] struct {
	n atomic.Int64
}

// Value returns the counter's value.
func (c *Counter[T]) Value() int64 {
	return c.n.Load()
}

// Func returns a Func[T, *Maybe[T]] that updates the counter.
func (c *Counter[T]) Func() Func[T, *Maybe[T]] {
	return &counterFunc[T]{c}
}

// counterFunc is the Func returned by CounterFunc.Func.
type counterFunc[T any] struct {
	c *Counter[T]
}

// Apply implements Func.
func (c *counterFunc[T]) Apply(ctx context.Context, value T) *Maybe[T] {
	c.c.n.Add(1)
	return &Maybe[T]{
		Error:        nil,
		Observations: nil,
		Skipped:      false,
		State:        value,
	}
}

// ErrorLogger logs errors emitted by Func[A, B].
type ErrorLogger struct {
	errors []error
	mu     sync.Mutex
}

// Errors returns the a copy of the internal array of errors and clears
// the internal array of errors as a side effect.
func (e *ErrorLogger) Errors() []error {
	defer e.mu.Unlock()
	e.mu.Lock()
	v := []error{}
	v = append(v, e.errors...)
	e.errors = nil // as documented
	return v
}

// Record records that an error occurred.
func (e *ErrorLogger) Record(err error) {
	defer e.mu.Unlock()
	e.mu.Lock()
	e.errors = append(e.errors, err)
}

// RecordErrors records errors returned by fx.
func RecordErrors[A, B any](logger *ErrorLogger, fx Func[A, *Maybe[B]]) Func[A, *Maybe[B]] {
	return &recordErrorsFunc[A, B]{
		fx: fx,
		p:  logger,
	}
}

// recordErrorsFunc is the type returned by ErrorLogger.Wrap.
type recordErrorsFunc[A, B any] struct {
	fx Func[A, *Maybe[B]]
	p  *ErrorLogger
}

// Apply implements Func.
func (elw *recordErrorsFunc[A, B]) Apply(ctx context.Context, a A) *Maybe[B] {
	r := elw.fx.Apply(ctx, a)
	if r.Error != nil {
		elw.p.Record(r.Error)
	}
	return r
}

// FirstErrorExcludingBrokenIPv6Errors returns the first error in a list of
// *Maybe[T] excluding errors known to be linked with IPv6 issues.
func FirstErrorExcludingBrokenIPv6Errors[T any](entries ...*Maybe[T]) error {
	for _, entry := range entries {
		if entry.Error != nil {
			continue
		}
		err := entry.Error
		switch err.Error() {
		case netxlite.FailureNetworkUnreachable, netxlite.FailureHostUnreachable:
			// This class of errors is often times linked with wrongly
			// configured IPv6, therefore we skip them.
		default:
			return err
		}
	}
	return nil
}

// FirstError returns the first error in a list of *Maybe[T].
func FirstError[T any](entries ...*Maybe[T]) error {
	for _, entry := range entries {
		if entry.Error != nil {
			continue
		}
		return entry.Error
	}
	return nil
}
