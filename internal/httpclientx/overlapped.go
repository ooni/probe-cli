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
	// RunFunc is the MANDATORY function that fetches the given [*Endpoint].
	//
	// This field is typically initialized by [NewOverlappedGetJSON], [NewOverlappedGetRaw],
	// [NewOverlappedGetXML], or [NewOverlappedPostJSON] to be the proper function that
	// makes sense for the operation that you requested with the constructor.
	//
	// If you set it manually, you MUST modify it before calling [*Overlapped.Run].
	RunFunc func(ctx context.Context, epnt *Endpoint) (Output, error)

	// ScheduleInterval is the MANDATORY scheduling interval.
	//
	// This field is typically initialized by [NewOverlappedGetJSON], [NewOverlappedGetRaw],
	// [NewOverlappedGetXML], or [NewOverlappedPostJSON] to be [OverlappedDefaultScheduleInterval].
	//
	// If you set it manually, you MUST modify it before calling [*Overlapped.Run].
	ScheduleInterval time.Duration
}

func newOverlappedWithFunc[Output any](fx func(context.Context, *Endpoint) (Output, error)) *Overlapped[Output] {
	return &Overlapped[Output]{
		RunFunc:          fx,
		ScheduleInterval: OverlappedDefaultScheduleInterval,
	}
}

// NewOverlappedGetJSON constructs a [*Overlapped] for calling [GetJSON] with multiple URLs.
func NewOverlappedGetJSON[Output any](config *Config) *Overlapped[Output] {
	return newOverlappedWithFunc(func(ctx context.Context, epnt *Endpoint) (Output, error) {
		return getJSON[Output](ctx, epnt, config)
	})
}

// NewOverlappedGetRaw constructs a [*Overlapped] for calling [GetRaw] with multiple URLs.
func NewOverlappedGetRaw(config *Config) *Overlapped[[]byte] {
	return newOverlappedWithFunc(func(ctx context.Context, epnt *Endpoint) ([]byte, error) {
		return getRaw(ctx, epnt, config)
	})
}

// NewOverlappedGetXML constructs a [*Overlapped] for calling [GetXML] with multiple URLs.
func NewOverlappedGetXML[Output any](config *Config) *Overlapped[Output] {
	return newOverlappedWithFunc(func(ctx context.Context, epnt *Endpoint) (Output, error) {
		return getXML[Output](ctx, epnt, config)
	})
}

// NewOverlappedPostJSON constructs a [*Overlapped] for calling [PostJSON] with multiple URLs.
func NewOverlappedPostJSON[Input, Output any](input Input, config *Config) *Overlapped[Output] {
	return newOverlappedWithFunc(func(ctx context.Context, epnt *Endpoint) (Output, error) {
		return postJSON[Input, Output](ctx, epnt, input, config)
	})
}

// ErrGenericOverlappedFailure indicates that a generic [*Overlapped] failure occurred.
var ErrGenericOverlappedFailure = errors.New("overlapped: generic failure")

// Run runs the overlapped operations, returning the result of the first operation
// that succeeds and otherwise returning an error describing what happened.
func (ovx *Overlapped[Output]) Run(ctx context.Context, epnts ...*Endpoint) (Output, error) {
	// create cancellable context for early cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// construct channel for collecting the results
	output := make(chan *erroror.Value[Output])

	// create ticker for scheduling subsequent attempts
	ticker := time.NewTicker(ovx.ScheduleInterval)
	defer ticker.Stop()

	// create index for the next endpoint to try
	idx := 0

	// create vector for collecting results
	results := []*erroror.Value[Output]{}

	// keep looping until we have results for each endpoints
	for len(results) < len(epnts) {

		// if possible, start and advance the index, when we've gone past
		// the index, we'll just keep waiting for channel events.
		if idx < len(epnts) {
			go ovx.transact(ctx, idx, epnts[idx], output)
			idx++
		}

		select {
		// this event means that a child goroutine completed
		// so we store the result and cancel the context on the
		// first success, to avoid doing duplicate work
		case result := <-output:
			results = append(results, result)
			if result.Err == nil {
				cancel()
			}

		// this means the ticker ticked, so we should loop again and
		// attempt another endpoint (if we've tried all of them ticking
		// is not going to hurt us anyway)
		case <-ticker.C:
		}
	}

	// postprocess the results to check for success and
	// aggregate all the errors that occurred
	errorv := []error{}
	for _, result := range results {
		if result.Err == nil {
			return result.Value, nil
		}
		errorv = append(errorv, result.Err)
	}

	// handle the case where there's no error
	if len(errorv) <= 0 {
		errorv = append(errorv, ErrGenericOverlappedFailure)
	}

	// return zero value and errors list
	return *new(Output), errors.Join(errorv...)
}

// transact performs an HTTP transaction with the given URL and writes results to the output channel.
func (ovx *Overlapped[Output]) transact(ctx context.Context, _ int, epnt *Endpoint, output chan<- *erroror.Value[Output]) {
	value, err := ovx.RunFunc(ctx, epnt)
	output <- &erroror.Value[Output]{Err: err, Value: value}
}
