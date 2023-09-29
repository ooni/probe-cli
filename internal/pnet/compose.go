package pnet

import "context"

// Compose composes two stages together.
func Compose[A, B, C any](s1 Stage[A, B], s2 Stage[B, C]) Stage[A, C] {
	return StageFunc[A, C](func(ctx context.Context, inputs <-chan Result[A], outputs chan<- Result[C]) {
		// create channel containing intermediate results
		intermediate := make(chan Result[B])

		// run the first stage in the background
		go s1.Run(ctx, inputs, intermediate)

		// run the second stage in the foreground
		s2.Run(ctx, intermediate, outputs)
	})
}

// Compose3 composes N=3 stages together.
func Compose3[A, B, C, D any](s1 Stage[A, B], s2 Stage[B, C], s3 Stage[C, D]) Stage[A, D] {
	return Compose(s1, Compose(s2, s3))
}
