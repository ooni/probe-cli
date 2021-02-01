package urlgetter

import (
	"context"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// MultiInput is the input for Multi.Run().
type MultiInput struct {
	// Config contains the configuration for this target.
	Config Config

	// Target contains the target URL to measure.
	Target string
}

// MultiOutput is the output returned by Multi.Run()
type MultiOutput struct {
	// Input is the input for which we measured.
	Input MultiInput

	// Err contains the measurement error.
	Err error

	// TestKeys contains the measured test keys.
	TestKeys TestKeys
}

// MultiGetter allows to override the behaviour of Multi for testing purposes.
type MultiGetter func(ctx context.Context, g Getter) (TestKeys, error)

// DefaultMultiGetter is the default MultiGetter
func DefaultMultiGetter(ctx context.Context, g Getter) (TestKeys, error) {
	return g.Get(ctx)
}

// Multi allows to run several urlgetters in paraller.
type Multi struct {
	// Begin is the time when the experiment begun. If you do not
	// set this field, every target is measured independently.
	Begin time.Time

	// Getter is the Getter func to be used. If this is nil we use
	// the default getter, which is what you typically want.
	Getter MultiGetter

	// Parallelism is the optional parallelism to be used. If this is
	// zero, or negative, we use a reasonable default.
	Parallelism int

	// Session is the session to be used. If this is nil, the Run
	// method will panic with a nil pointer error.
	Session model.ExperimentSession
}

// Run performs several urlgetters in parallel. This function returns a channel
// where each result is posted. This function will always perform all the requested
// measurements: if the ctx is canceled or its deadline expires, then you will see
// a bunch of failed measurements. Since all measurements are always performed,
// you know you're done when you've read len(inputs) results in output.
func (m Multi) Run(ctx context.Context, inputs []MultiInput) <-chan MultiOutput {
	parallelism := m.Parallelism
	if parallelism <= 0 {
		const defaultParallelism = 3
		parallelism = defaultParallelism
	}
	inputch := make(chan MultiInput)
	outputch := make(chan MultiOutput)
	go m.source(inputs, inputch)
	for i := 0; i < parallelism; i++ {
		go m.do(ctx, inputch, outputch)
	}
	return outputch
}

// Collect prints on the output channel the result of running urlgetter
// on every provided input. It closes the output channel when done.
func (m Multi) Collect(ctx context.Context, inputs []MultiInput,
	prefix string, callbacks model.ExperimentCallbacks) <-chan MultiOutput {
	return m.CollectOverall(ctx, inputs, 0, len(inputs), prefix, callbacks)
}

// CollectOverall prints on the output channel the result of running urlgetter
// on every provided input. You can use this method if you perform multiple collection
// tasks within one experiment as it allows to calculate the overall progress correctly
func (m Multi) CollectOverall(ctx context.Context, inputChunk []MultiInput, overallStartIndex int, overallCount int,
	prefix string, callbacks model.ExperimentCallbacks) <-chan MultiOutput {
	outputch := make(chan MultiOutput)
	go m.collect(len(inputChunk), overallStartIndex, overallCount, prefix, callbacks, m.Run(ctx, inputChunk), outputch)
	return outputch
}

// collect drains inputch, prints progress, and emits to outputch. When done, this
// function will close outputch to notify the calller.
func (m Multi) collect(expect int, overallStartIndex int, overallCount int, prefix string, callbacks model.ExperimentCallbacks,
	inputch <-chan MultiOutput, outputch chan<- MultiOutput) {
	count := overallStartIndex
	var index int
	defer close(outputch)
	for index < expect {
		entry := <-inputch
		index++
		count++
		percentage := float64(count) / float64(overallCount)
		callbacks.OnProgress(percentage, fmt.Sprintf(
			"%s: measure %s: %+v", prefix, entry.Input.Target, entry.Err,
		))
		outputch <- entry
	}
}

// source posts all the inputs in the inputch. When done, this
// method will close the input channel to notify the reader.
func (m Multi) source(inputs []MultiInput, inputch chan<- MultiInput) {
	defer close(inputch)
	for _, input := range inputs {
		inputch <- input
	}
}

// do performs urlgetter on all the inputs read from the in channel and
// writes the results on the out channel. If the context is canceled, or
// its deadline expires, this function will continue performing all the
// required measurements, which will all fail.
func (m Multi) do(ctx context.Context, in <-chan MultiInput, out chan<- MultiOutput) {
	for input := range in {
		g := Getter{
			Begin:   m.Begin,
			Config:  input.Config,
			Session: m.Session,
			Target:  input.Target,
		}
		fn := m.Getter
		if fn == nil {
			fn = DefaultMultiGetter
		}
		tk, err := fn(ctx, g)
		out <- MultiOutput{Input: input, Err: err, TestKeys: tk}
	}
}
