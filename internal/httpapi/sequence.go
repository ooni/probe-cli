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

// SequenceCaller calls the API specified by |Descriptor| once for each of
// the available |Endpoints| until one of them succeeds.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
type SequenceCaller struct {
	// Descriptor is the API |Descriptor|.
	Descriptor *Descriptor

	// Endpoints is the list of |Endpoint| to use.
	Endpoints []*Endpoint
}

// NewSequenceCaller is a factory for creating a |SequenceCaller|.
func NewSequenceCaller(desc *Descriptor, endpoints ...*Endpoint) *SequenceCaller {
	return &SequenceCaller{
		Descriptor: desc,
		Endpoints:  endpoints,
	}
}

// ErrAllEndpointsFailed indicates that all endpoints failed.
var ErrAllEndpointsFailed = errors.New("httpapi: all endpoints failed")

// shouldRetry returns true when we should try with another endpoint given the
// value of |err| which could (obviously) be nil in case of success.
func (sc *SequenceCaller) shouldRetry(err error) bool {
	var kind *errMaybeCensorship
	belongs := errors.As(err, &kind)
	return belongs
}

// Call calls |Call| for each |Endpoint| and |Descriptor| until one endpoint succeeds. The
// return value is the response body and the selected endpoint index or the error.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
func (sc *SequenceCaller) Call(ctx context.Context) ([]byte, int, error) {
	var selected int
	merr := multierror.New(ErrAllEndpointsFailed)
	for _, epnt := range sc.Endpoints {
		respBody, err := Call(ctx, sc.Descriptor, epnt)
		if sc.shouldRetry(err) {
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

// CallWithJSONResponse is like |SequenceCaller.Call| except that it invokes the
// underlying |CallWithJSONResponse| rather than invoking |Call|.
//
// CAVEAT: this code will ONLY retry API calls with subsequent endpoints when
// the error originates in the HTTP round trip or while reading the body.
func (sc *SequenceCaller) CallWithJSONResponse(ctx context.Context, response any) (int, error) {
	var selected int
	merr := multierror.New(ErrAllEndpointsFailed)
	for _, epnt := range sc.Endpoints {
		err := CallWithJSONResponse(ctx, sc.Descriptor, epnt, response)
		if sc.shouldRetry(err) {
			merr.Add(err)
			selected++
			continue
		}
		// Note: some errors will lead us to return
		// early as documented for this method
		return selected, err
	}
	return -1, merr
}
