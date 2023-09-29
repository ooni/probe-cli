package pnet

import "context"

// Run runs a [Stage] with the given input and collects its outputs.
func Run[A, B any](ctx context.Context, stage Stage[A, B], input A) []Result[B] {
	// create buffered channel for sending input
	inputs := make(chan Result[A], 1)

	// create outputs channel
	outputs := make(chan Result[B])

	// start the pipeline
	go stage.Run(ctx, inputs, outputs)

	// send input to the pipeline
	inputs <- NewResultValue(input)

	// tell the pipeline there won't be any more input
	close(inputs)

	// collect all the results together
	return Collect(outputs)
}
