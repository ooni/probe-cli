package session

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
func (s *Session) checkin(ctx context.Context, req *Request) {
	result, err := s.docheckin(ctx, req)
	event := &Event{
		CheckIn: &CheckInEvent{
			Error:  err,
			Result: result,
		},
	}
	s.emit(event)
}

// docheckin implements checkin.
func (s *Session) docheckin(ctx context.Context, req *Request) (*model.OOAPICheckInResult, error) {
	runtimex.Assert(req.CheckIn != nil, "passed a nil CheckIn")

	if s.state == nil {
		return nil, ErrNotBootstrapped
	}

	ts := newTickerService(ctx, s)
	defer ts.stop()

	result, err := s.state.backendClient.CheckIn(ctx, req.CheckIn)
	if err != nil {
		return nil, err
	}
	s.state.checkIn = result
	return result, nil
}
