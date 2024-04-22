package httpclientx

//
// overlapped.go - overlapped operations.
//

import (
	"context"
	"errors"
	"time"

	"github.com/ooni/probe-cli/v3/internal/erroror"
)

// TODO(bassosimone): we still need to write tests for overlapped.

// OverlappedDefaultScheduleInterval is the default schedule interval. After this interval
// has elapsed for a URL without seeing a success, we will schedule the next URL.
const OverlappedDefaultScheduleInterval = 15 * time.Second

// Overlapped represents the possibility of overlapping HTTP calls for a set of
// functionally equivalent URLs, such that we start a new call if the previous one
// has failed to produce a result within the configured ScheduleInterval.
//
// # Limitations
//
// Under very bad networking conditions, [*Overlapped] would cause a new network
// call to start while the previous one is still in progress and very slowly downloading
// a response. A future implementation SHOULD probably account for this possibility.
type Overlapped[Output any] struct {
	// RunFunc is the MANDATORY function that fetches the given URL.
	RunFunc func(ctx context.Context, URL string) (Output, error)

	// ScheduleInterval is the MANDATORY scheduling interval.
	ScheduleInterval time.Duration
}

// NewOverlappedGetJSON constructs a [*Overlapped] for calling [GetJSON].
func NewOverlappedGetJSON[Output any](ctx context.Context, config *Config) *Overlapped[Output] {
	return &Overlapped[Output]{
		RunFunc: func(ctx context.Context, URL string) (Output, error) {
			return GetJSON[Output](ctx, config, URL)
		},
		ScheduleInterval: OverlappedDefaultScheduleInterval,
	}
}

// NewOverlappedGetRaw constructs a [*Overlapped] for calling [GetRaw].
func NewOverlappedGetRaw(ctx context.Context, config *Config) *Overlapped[[]byte] {
	return &Overlapped[[]byte]{
		RunFunc: func(ctx context.Context, URL string) ([]byte, error) {
			return GetRaw(ctx, config, URL)
		},
		ScheduleInterval: OverlappedDefaultScheduleInterval,
	}
}

// NewOverlappedGetXML constructs a [*Overlapped] for calling [GetXML].
func NewOverlappedGetXML[Output any](ctx context.Context, config *Config) *Overlapped[Output] {
	return &Overlapped[Output]{
		RunFunc: func(ctx context.Context, URL string) (Output, error) {
			return GetXML[Output](ctx, config, URL)
		},
		ScheduleInterval: OverlappedDefaultScheduleInterval,
	}
}

// NewOverlappedPostJSON constructs a [*Overlapped] for calling [PostJSON].
func NewOverlappedPostJSON[Input, Output any](ctx context.Context, config *Config, input Input) *Overlapped[Output] {
	return &Overlapped[Output]{
		RunFunc: func(ctx context.Context, URL string) (Output, error) {
			return PostJSON[Input, Output](ctx, config, URL, input)
		},
		ScheduleInterval: OverlappedDefaultScheduleInterval,
	}
}

// ErrGenericOverlappedFailure indicates that a generic [*Overlapped] failure occurred.
var ErrGenericOverlappedFailure = errors.New("overlapped: generic failure")

// Run runs the overlapped operations, returning the result of the first operation
// that succeeds and otherwise returning an error describing what happened.
//
// # Limitations
//
// This implementation creates a new goroutine for each provided URL under the assumption that
// the overall number of URLs is small. A future revision would address this issue.
func (ovx *Overlapped[Output]) Run(ctx context.Context, URLs ...string) (Output, error) {
	// create cancellable context for early cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// construct channel for collecting the results
	output := make(chan *erroror.Value[Output])

	// schedule a measuring goroutine per URL.
	for idx := 0; idx < len(URLs); idx++ {
		go ovx.transact(ctx, idx, URLs[idx], output)
	}

	// we expect to see exactly a response for each goroutine
	var (
		firstOutput *Output
		errorv      []error
	)
	for idx := 0; idx < len(URLs); idx++ {
		// get a result from one of the goroutines
		result := <-output

		// handle the error case
		if result.Err != nil {
			errorv = append(errorv, result.Err)
			continue
		}

		// possibly record the first success
		if firstOutput == nil {
			firstOutput = &result.Value
		}

		// make sure we interrupt all the other goroutines
		cancel()
	}

	// handle the case of success
	if firstOutput != nil {
		return *firstOutput, nil
	}

	// handle the case where there's no error
	if len(errorv) <= 0 {
		errorv = append(errorv, ErrGenericOverlappedFailure)
	}

	// return zero value and errors list
	return *new(Output), errors.Join(errorv...)
}

// transact performs an HTTP transaction with the given URL and writes results to the output channel.
func (ovx *Overlapped[Output]) transact(ctx context.Context, idx int, URL string, output chan<- *erroror.Value[Output]) {
	// wait for our time to start
	//
	// add one nanosecond to make sure the delay is always positive
	timer := time.NewTimer(time.Duration(idx)*ovx.ScheduleInterval + time.Nanosecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		output <- &erroror.Value[Output]{Err: ctx.Err()}
		return
	case <-timer.C:
		// fallthrough
	}

	// obtain the results
	value, err := ovx.RunFunc(ctx, URL)

	// emit the results
	output <- &erroror.Value[Output]{Err: err, Value: value}
}
