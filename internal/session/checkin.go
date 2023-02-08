package session

//
// Code to call the check-in API.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// CheckInRequest is the request for the check-in API.
type CheckInRequest = model.OOAPICheckInConfig

// CheckInEvent is the result of calling the check-in API.
type CheckInEvent struct {
	// Error is the geolocate result.
	Error error

	// Result is the result returned on success.
	Result *model.OOAPICheckInResult
}

// checkin calls the check-in API.
func (s *Session) checkin(ctx context.Context, req *CheckInRequest) {
	runtimex.Assert(req != nil, "passed a nil req")
	result, err := s.docheckin(ctx, req)
	s.maybeEmit(&Event{
		CheckIn: &CheckInEvent{
			Error:  err,
			Result: result,
		},
	})
}

// docheckin implements checkin.
func (s *Session) docheckin(ctx context.Context, req *CheckInRequest) (*model.OOAPICheckInResult, error) {
	if s.state.IsNone() {
		return nil, ErrNotBootstrapped
	}

	ts := newTickerService(ctx, s)
	defer ts.stop()

	result, err := s.state.Unwrap().backendClient.CheckIn(ctx, req)
	if err != nil {
		return nil, err
	}
	s.state.Unwrap().checkIn = model.NewOptionalPtr(result)
	return result, nil
}
