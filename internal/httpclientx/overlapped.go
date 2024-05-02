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

	// Semaphore is the MANDATORY channel working as a semaphore for cross-signaling
	// between goroutines such that we don't wait the full ScheduleInterval if an
	// attempt that was previously scheduled failed very early.
	//
	// This field is typically initialized by [NewOverlappedGetJSON], [NewOverlappedGetRaw],
	// [NewOverlappedGetXML], or [NewOverlappedPostJSON] using [NewOverlappedSemaphore].
	//
	// If you set it manually, initialize it with [NewOverlappedSemaphore] as follows:
	//
	//	overlapped.Semaphore = NewOverlappedSemaphore()
	//
	// Also, you MUST initialize this field before calling [*Overlapped.Run].
	Semaphore chan any
}

// NewOverlappedSemaphore properly initializes a semaphore for the [*Overlapped] struct.
func NewOverlappedSemaphore() (out chan any) {
	out = make(chan any, 1)
	out <- true
	return
}

func newOverlappedWithFunc[Output any](fx func(context.Context, *Endpoint) (Output, error)) *Overlapped[Output] {
	return &Overlapped[Output]{
		RunFunc:          fx,
		ScheduleInterval: OverlappedDefaultScheduleInterval,
		Semaphore:        NewOverlappedSemaphore(),
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
//
// # Limitations
//
// This implementation creates a new goroutine for each provided URL under the assumption that
// the overall number of URLs is small. A future revision would address this issue.
func (ovx *Overlapped[Output]) Run(ctx context.Context, epnts ...*Endpoint) (Output, error) {
	// create cancellable context for early cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// construct channel for collecting the results
	output := make(chan *erroror.Value[Output])

	// schedule a measuring goroutine per URL.
	for idx := 0; idx < len(epnts); idx++ {
		go ovx.transact(ctx, idx, epnts[idx], output)
	}

	// we expect to see exactly a response for each goroutine
	var (
		firstOutput *Output
		errorv      []error
	)
	for idx := 0; idx < len(epnts); idx++ {
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
func (ovx *Overlapped[Output]) transact(ctx context.Context, idx int, epnt *Endpoint, output chan<- *erroror.Value[Output]) {
	// wait for our time to start
	//
	// add one nanosecond to make sure the delay is always positive
	timer := time.NewTimer(time.Duration(idx)*ovx.ScheduleInterval + time.Nanosecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		output <- &erroror.Value[Output]{Err: ctx.Err()}
		return
	case <-ovx.Semaphore:
		// fallthrough
	case <-timer.C:
		// fallthrough
	}

	// obtain the results
	value, err := ovx.RunFunc(ctx, epnt)

	// emit the results
	output <- &erroror.Value[Output]{Err: err, Value: value}

	// unblock the next goroutine
	ovx.Semaphore <- true
}
