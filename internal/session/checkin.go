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
	if s.state.Unwrap().checkIn.IsSome() {
		// TODO(bassosimone): in the future we should define caching
		// policies for the check-in response, but for now this is fine.
		return s.state.Unwrap().checkIn.Unwrap(), nil
	}

	ts := newTickerService(ctx, s)
	defer ts.stop()

	backendClient := s.state.Unwrap().backendClient
	result, err := backendClient.CheckIn(ctx, req)
	if err != nil {
		return nil, err
	}
	s.state.Unwrap().checkIn = model.NewOptionalPtr(result)
	return result, nil
}
