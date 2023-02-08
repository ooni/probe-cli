package session

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
func (s *Session) submit(ctx context.Context, req *Request) {
	s.emit(&Event{
		Submit: &SubmitEvent{
			Error: s.dosubmit(ctx, req),
		},
	})
}

// dosubmit implements submit.
func (s *Session) dosubmit(ctx context.Context, req *Request) error {
	runtimex.Assert(req.Submit != nil, "passed a nil Submit")

	if s.state == nil {
		return ErrNotBootstrapped
	}

	ts := newTickerService(ctx, s)
	defer ts.stop()

	return s.state.backendClient.Submit(ctx, req.Submit)
}
