package session

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// GeolocateRequest contains config for geolocate.
type GeolocateRequest struct{}

// GeolocateEvent is the event emitted at the end of geolocate.
type GeolocateEvent struct {
	// Error is the geolocate result.
	Error error

	// Location is the probe location.
	Location *geolocate.Results
}

// geolocate performs a geolocation.
func (s *Session) geolocate(ctx context.Context, req *Request) {
	location, err := s.dogeolocate(ctx, req)
	event := &Event{
		Geolocate: &GeolocateEvent{
			Error:    err,
			Location: location,
		},
	}
	s.emit(event)
}

// ErrNotBootstrapped indicates we didn't bootstrap a session.
var ErrNotBootstrapped = errors.New("session: not bootstrapped")

// dogeolocate implements geolocate.
func (s *Session) dogeolocate(ctx context.Context, req *Request) (*geolocate.Results, error) {
	runtimex.Assert(req.Geolocate != nil, "passed a nil Geolocate")

	if s.state == nil {
		return nil, ErrNotBootstrapped
	}

	ts := newTickerService(ctx, s)
	defer ts.stop()

	geolocateConfig := geolocate.Config{
		Resolver:  s.state.resolver,
		Logger:    s.state.logger,
		UserAgent: model.HTTPHeaderUserAgent,
	}
	task := geolocate.NewTask(geolocateConfig) // XXX

	// TODO(bassosimone): we should make geolocate.Results a type
	// in the internal/model package and use the ~typical naming for
	// its fields rather than the naming we have here.
	location, err := task.Run(ctx)
	if err != nil {
		return nil, err
	}

	s.state.location = location
	return location, nil
}
