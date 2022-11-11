package httpapi

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/multierror"
)

// SequenceCaller calls the API specified by |Descriptor| once for each
// available |Endpoints| until one of them succeds or all fail.
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

// Call calls |Call| for each |Endpoint| and |Descriptor| until one endpoint succeeds. The
// return value is the response body and the selected endpoint index or the error.
func (sc *SequenceCaller) Call(ctx context.Context) ([]byte, int, error) {
	var selected int
	merr := multierror.New(ErrAllEndpointsFailed)
	for _, epnt := range sc.Endpoints {
		respBody, err := Call(ctx, sc.Descriptor, epnt)
		if err != nil {
			merr.Add(err)
			selected++
			continue
		}
		return respBody, selected, nil
	}
	return nil, -1, merr
}

// CallWithJSONResponse calls |CallWithJSONResponse| for each |Endpoint|
// and |Descriptor| until one endpoint succeeds. The return value is
// the selected endpoint index or the error that occurred.
func (sc *SequenceCaller) CallWithJSONResponse(ctx context.Context, response any) (int, error) {
	var selected int
	merr := multierror.New(ErrAllEndpointsFailed)
	for _, epnt := range sc.Endpoints {
		if err := CallWithJSONResponse(ctx, sc.Descriptor, epnt, response); err != nil {
			merr.Add(err)
			selected++
			continue
		}
		return selected, nil
	}
	return -1, merr
}
