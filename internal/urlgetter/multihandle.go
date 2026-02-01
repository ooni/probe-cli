package urlgetter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/erroror"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// MultiResult is a measurement result returned by [*MultiHandle].
type MultiResult struct {
	// Target is the target we measured.
	Target *EasyTarget

	// TestKeys contains the [*TestKeys] or an error.
	TestKeys erroror.Value[*TestKeys]
}

// MultiHandle allows to run several measurements in paraller.
//
// The zero value is invalid. Please, initialize the MANDATORY fields.
type MultiHandle struct {
	// Begin is the OPTIONAL time when the experiment begun. If you do not
	// set this field, every target is measured independently.
	Begin time.Time

	// IndexGen is the MANDATORY index generator.
	IndexGen RunnerTraceIndexGenerator

	// Parallelism is the OPTIONAL parallelism to use. If this is
	// zero, or negative, we use a reasonable default.
	Parallelism int

	// Session is the MANDATORY session to use. If this is nil, the Run
	// method will panic with a nil pointer error.
	Session RunnerSession

	// UNet is the OPTIONAL underlying networ to be use.
	UNet model.UnderlyingNetwork
}

// Run measures the given targets in parallel using background goroutines and
// returns a channel where we post the measurement results.
func (hx *MultiHandle) Run(ctx context.Context, targets ...*EasyTarget) <-chan *MultiResult {
	// determine the parallelism to use
	const defaultParallelism = 3
	parallelism := max(hx.Parallelism, defaultParallelism)

	// create output channel
	output := make(chan *MultiResult)

	// create input channel
	input := make(chan *EasyTarget, len(targets))

	// emit the targets
	go func() {
		defer close(input)
		for _, target := range targets {
			input <- target
		}
	}()

	// create wait group for awaiting for workers to be done
	wg := &sync.WaitGroup{}

	// run workers in parallel
	for idx := 0; idx < parallelism; idx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hx.worker(ctx, input, output)
		}()
	}

	// close the output channel when all workers are done
	go func() {
		defer close(output)
		wg.Wait()
	}()

	// return the output channel to the caller
	return output
}

// worker is a worker used by [*MultiHandle] to perform parallel work.
func (hx *MultiHandle) worker(ctx context.Context, input <-chan *EasyTarget, output chan<- *MultiResult) {
	// process each target in the input stream
	for target := range input {

		// initialize the result with empty TestKeys field
		res := &MultiResult{
			Target:   target,
			TestKeys: erroror.Value[*TestKeys]{},
		}

		// create the easy handle
		easy := &EasyHandle{
			Begin:    hx.Begin,
			IndexGen: hx.IndexGen,
			Session:  hx.Session,
			UNet:     hx.UNet,
		}

		// perform the actual measurement
		testkeys, err := easy.Run(ctx, target)

		// assign either the error or the test keys
		if err != nil {
			res.TestKeys.Err = err
		} else {
			res.TestKeys.Value = testkeys
		}

		// emit the measurement result
		output <- res
	}
}

// MultiCollect is a filter that reads measurements from the results channel, prints
// progress using the given callbacks, and post measurements on the returned channel.
//
// # Arguments
//
// - callbacks contains the experiment callbacks used to print progress.
//
// - overallStartIndex is the index from which we should start
// counting for printing progress, typically 0.
//
// - overallTotal is the total number of entries we're measuring. If this
// value is zero or negative, we assume a total count of 1.
//
// - prefix is the prefix to use for printing progress.
//
// - results is the channel from which to read measurement results.
func MultiCollect(
	callbacks model.ExperimentCallbacks,
	overallStartIndex int,
	overallTotal int,
	prefix string,
	results <-chan *MultiResult,
) <-chan *MultiResult {
	// create output channel
	output := make(chan *MultiResult)

	// process the results channel in the background
	go func() {
		// make sure we close output when done
		defer close(output)

		// initialize count to be the specified start index
		count := overallStartIndex

		// process each entry
		for result := range results {
			// increment the number of results seen
			count++

			// emit progress information
			prog := multiComputeProgress(count, overallTotal)
			callbacks.OnProgress(prog, fmt.Sprintf(
				"%s: measure %s: %+v",
				prefix,
				result.Target.URL,
				model.ErrorToStringOrOK(result.TestKeys.Err),
			))

			// emit the result entry
			output <- result
		}
	}()

	// return the output channel to the caller
	return output
}

// multiComputeProgress computes the progress trying to avoid divide by zero and
// returning values greater than the maximum multiComputeProgress value.
func multiComputeProgress(count, overallTotal int) float64 {
	return min(1.0, float64(count)/float64(max(overallTotal, 1)))
}
