package session

//
// Submitting measurements
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// SubmitRequest is the request for the measurement submission API.
type SubmitRequest = model.Measurement

// SubmitEvent is the result of submitting a measurement.
type SubmitEvent struct {
	// Error is the geolocate result.
	Error error
}

// submit submits a measurement.
func (s *Session) submit(ctx context.Context, req *SubmitRequest) {
	s.maybeEmit(&Event{
		Submit: &SubmitEvent{
			Error: s.dosubmit(ctx, req),
		},
	})
}

// dosubmit implements submit.
func (s *Session) dosubmit(ctx context.Context, req *SubmitRequest) error {
	runtimex.Assert(req != nil, "passed a nil req")

	if s.state.IsNone() {
		return ErrNotBootstrapped
	}

	ts := newTickerService(ctx, s)
	defer ts.stop()

	return s.state.Unwrap().backendClient.Submit(ctx, req)
}
