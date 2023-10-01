package pnet

import (
	"context"
)

// Stage is a pipeline stage that reads its input channel containing requests and
// writes results on its output channel. An upstream [Stage] closes its input channel
// to notify a [Stage] that no further requests would come. When a [Stage] sees that
// its input channel has been closed, it closes it output channel and returns.
type Stage[A, B any] interface {
	Run(ctx context.Context, inputs <-chan Result[A], outputs chan<- Result[B])
}

// StageFunc adapts a func to be a [Stage].
type StageFunc[A, B any] func(ctx context.Context, inputs <-chan Result[A], outputs chan<- Result[B])

// Run implements [Stage].
func (fx StageFunc[A, B]) Run(ctx context.Context, inputs <-chan Result[A], outputs chan<- Result[B]) {
	fx(ctx, inputs, outputs)
}

// stageForAction creates a [Stage] for a given action
func stageForAction[A, B any](act action[A, B]) Stage[A, B] {
	return StageFunc[A, B](func(ctx context.Context, inputs <-chan Result[A], outputs chan<- Result[B]) {
		defer close(outputs)
		for input := range inputs {
			if err := input.Err; err != nil {
				outputs <- NewResultError[B](err)
				continue
			}
			act.Run(ctx, input.Value, outputs)
		}
	})
}
