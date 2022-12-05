package httpapi

//
// Sequentially call available API endpoints until one succeed
// or all of them fail. A future implementation of this code may
// (probably should?) take into account knowledge of what is
// working and what is not working to optimize the order with
// which to try different alternatives.
//

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/multierror"
)

// ErrAllEndpointsFailed indicates that all endpoints failed.
var ErrAllEndpointsFailed = errors.New("httpapi: all endpoints failed")

// sequenceCallerShouldRetry returns true when we should try with another endpoint
// given the value of |err| which could (obviously) be nil in case of success.
func sequenceCallerShouldRetry(err error) bool {
	var kind *errMaybeCensorship
	belongs := errors.As(err, &kind)
	return belongs
}

// SimpleSequenceCaller allows to call a given API with several endpoints.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
type SimpleSequenceCaller struct {
	// Spec is the API spec
	Spec SimpleSpec

	// Endpoints is the list of endpoints
	Endpoints []*Endpoint
}

// NewSimpleSequenceCaller is a factory for creating a SimpleSequenceCaller.
func NewSimpleSequenceCaller(spec SimpleSpec, endpoints ...*Endpoint) *SimpleSequenceCaller {
	return &SimpleSequenceCaller{
		Spec:      spec,
		Endpoints: endpoints,
	}
}

// SimpleCall calls the api represented by ssc.Spec for each of the ssc.Endpoints until either
// we succeed or all the endpoints have failed. The return value is the response body
// and the selected endpoint index, on success, or the error, on failure.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
func (ssc *SimpleSequenceCaller) SimpleCall(ctx context.Context) ([]byte, int, error) {
	var selected int
	merr := multierror.New(ErrAllEndpointsFailed)
	for _, epnt := range ssc.Endpoints {
		respBody, err := RawCall(ctx, ssc.Spec, epnt)
		if sequenceCallerShouldRetry(err) {
			merr.Add(err)
			selected++
			continue
		}
		// Note: some errors will lead us to return
		// early as documented for this method
		return respBody, selected, err
	}
	return nil, -1, merr
}

// TypedSequenceCaller allows to call a given API with several endpoints.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
type TypedSequenceCaller[T any] struct {
	// Spec is the API spec
	Spec TypedSpec[T]

	// Endpoints is the list of endpoints
	Endpoints []*Endpoint
}

// NewTypedSequenceCaller is a factory for creating a TypedSequenceCaller.
func NewTypedSequenceCaller[T any](spec TypedSpec[T], endpoints ...*Endpoint) *TypedSequenceCaller[T] {
	return &TypedSequenceCaller[T]{
		Spec:      spec,
		Endpoints: endpoints,
	}
}

// TypedCall calls the api represented by ssc.Spec for each of the ssc.Endpoints until either
// we succeed or all the endpoints have failed. The return value is the response body
// and the selected endpoint index, on success, or the error, on failure.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
func (tsc *TypedSequenceCaller[T]) TypedCall(ctx context.Context) (*T, int, error) {
	var selected int
	merr := multierror.New(ErrAllEndpointsFailed)
	for _, epnt := range tsc.Endpoints {
		value, err := TypedCall(ctx, tsc.Spec, epnt)
		if sequenceCallerShouldRetry(err) {
			merr.Add(err)
			selected++
			continue
		}
		// Note: some errors will lead us to return
		// early as documented for this method
		return value, selected, err
	}
	return nil, -1, merr
}
