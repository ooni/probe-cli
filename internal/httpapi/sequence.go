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
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// SequenceCaller calls the API specified by [Descriptor] once for each of
// the available [Endpoint]s until one of them succeeds.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
type SequenceCaller[RequestType, ResponseType any] struct {
	// Descriptor is the API [Descriptor].
	Descriptor *Descriptor[RequestType, ResponseType]

	// Endpoints is the list of [Endpoint] to use.
	Endpoints []*Endpoint
}

// NewSequenceCaller is a factory for creating a [SequenceCaller].
func NewSequenceCaller[RequestType, ResponseType any](
	desc *Descriptor[RequestType, ResponseType],
	endpoints ...*Endpoint,
) *SequenceCaller[RequestType, ResponseType] {
	return &SequenceCaller[RequestType, ResponseType]{
		Descriptor: desc,
		Endpoints:  endpoints,
	}
}

// ErrAllEndpointsFailed indicates that all endpoints failed.
var ErrAllEndpointsFailed = errors.New("httpapi: all endpoints failed")

// sequenceCallershouldRetry returns true when we should try with another endpoint given the
// value of err which could (obviously) be nil in case of success.
func sequenceCallerShouldRetry(err error) bool {
	var kind *errMaybeCensorship
	belongs := errors.As(err, &kind)
	return belongs
}

// Call calls [Call] for each [Endpoint] and [Descriptor] until one endpoint succeeds. The
// return value is the response body and the selected endpoint index or the error.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
func (sc *SequenceCaller[RequestType, ResponseType]) Call(ctx context.Context) (ResponseType, int, error) {
	runtimex.Assert(sc.Descriptor.Response != nil, "sc.Descriptor.Response is nil")
	var selected int
	merr := multierror.New(ErrAllEndpointsFailed)
	for _, epnt := range sc.Endpoints {
		respBody, err := Call(ctx, sc.Descriptor, epnt)
		if sequenceCallerShouldRetry(err) {
			merr.Add(err)
			selected++
			continue
		}
		// Note: some errors will lead us to return
		// early as documented for this method
		return respBody, selected, err
	}
	return *new(ResponseType), -1, merr
}
