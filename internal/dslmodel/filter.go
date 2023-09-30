package dslmodel

import (
	"context"
)

// Filter transforms a [Result] wrapped type A into a [Result] wrapped type B.
type Filter[A, B any] interface {
	Run(ctx context.Context, rt Runtime, minput Result[A]) Result[B]
}

// FilterFunc converts a func into a [Filter].
type FilterFunc[A, B any] func(ctx context.Context, rt Runtime, minput Result[A]) Result[B]

// Run implements Filter.
func (f FilterFunc[A, B]) Run(ctx context.Context, rt Runtime, minput Result[A]) Result[B] {
	return f(ctx, rt, minput)
}

// FilterToPipeline converts a [Filter] to a [Pipeline].
func FilterToPipeline[A, B any](f Filter[A, B]) Pipeline[A, B] {
	return PipelineFunc[A, B](func(ctx context.Context, rt Runtime, inputs <-chan Result[A]) <-chan Result[B] {
		outputs := make(chan Result[B])

		go func() {
			defer close(outputs)
			for input := range inputs {
				outputs <- f.Run(ctx, rt, input)
			}
		}()

		return outputs
	})
}
