package dslmodel

import "context"

// Generator transforms a [Result] wrapped type A into a stream of [Result] wrapped type B.
type Generator[A, B any] interface {
	Run(ctx context.Context, rt Runtime, minput Result[A]) <-chan Result[B]
}

// GeneratorFunc converts a func into a [Generator].
type GeneratorFunc[A, B any] func(ctx context.Context, rt Runtime, minput Result[A]) <-chan Result[B]

// Run implements Generator.
func (f GeneratorFunc[A, B]) Run(ctx context.Context, rt Runtime, minput Result[A]) <-chan Result[B] {
	return f(ctx, rt, minput)
}

// GeneratorToPipeline converts a [Generator] to a [Pipeline].
func GeneratorToPipeline[A, B any](g Generator[A, B]) Pipeline[A, B] {
	return PipelineFunc[A, B](func(ctx context.Context, rt Runtime, inputs <-chan Result[A]) <-chan Result[B] {
		outputs := make(chan Result[B])

		go func() {
			defer close(outputs)
			for input := range inputs {
				for output := range g.Run(ctx, rt, input) {
					outputs <- output
				}
			}
		}()

		return outputs
	})
}
