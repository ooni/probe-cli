package dslmodel

import "context"

// Pipeline is a measurement pipeline transforming [Result] wrapped A to [Result] wrapped B.
type Pipeline[A, B any] interface {
	Run(ctx context.Context, rt Runtime, inputs <-chan Result[A]) <-chan Result[B]
}

// PipelineFunc converts a func into a [Pipeline].
type PipelineFunc[A, B any] func(ctx context.Context, rt Runtime, inputs <-chan Result[A]) <-chan Result[B]

// Run implements Pipeline.
func (f PipelineFunc[A, B]) Run(ctx context.Context, rt Runtime, inputs <-chan Result[A]) <-chan Result[B] {
	return f(ctx, rt, inputs)
}

// ComposePipelines composes two pipelines together to create a more complex pipeline.
func ComposePipelines[A, B, C any](p1 Pipeline[A, B], p2 Pipeline[B, C]) Pipeline[A, C] {
	return PipelineFunc[A, C](func(ctx context.Context, rt Runtime, inputs <-chan Result[A]) <-chan Result[C] {
		return p2.Run(ctx, rt, p1.Run(ctx, rt, inputs))
	})
}

// ComposePipelines3 composes three pipelines together to create a more complex pipeline.
func ComposePipelines3[A, B, C, D any](p1 Pipeline[A, B], p2 Pipeline[B, C], p3 Pipeline[C, D]) Pipeline[A, D] {
	return ComposePipelines(p1, ComposePipelines(p2, p3))
}
