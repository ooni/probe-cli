package pnet

import "context"

// action is the implementation of [Stage] that takes in input a result's value
// and emits on the output channel the corresponding output.
type action[A, B any] interface {
	Run(ctx context.Context, input A, outputs chan<- Result[B])
}

// actionFunc is an adapter to convert a func to an become an [Action].
type actionFunc[A, B any] func(ctx context.Context, input A, outputs chan<- Result[B])

// Run implements Action.
func (fx actionFunc[A, B]) Run(ctx context.Context, input A, outputs chan<- Result[B]) {
	fx(ctx, input, outputs)
}
