package session

//
// Geolocating a probe.
//

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

	// Location is the geolocated location.
	Location *geolocate.Results
}

// geolocate performs a geolocation.
func (s *Session) geolocate(ctx context.Context, req *GeolocateRequest) {
	runtimex.Assert(req != nil, "passed a nil req")
	location, err := s.dogeolocate(ctx, req)
	s.maybeEmit(&Event{
		Geolocate: &GeolocateEvent{
			Error:    err,
			Location: location,
		},
	})
}

// ErrNotBootstrapped indicates we have not bootstrapped the session yet.
var ErrNotBootstrapped = errors.New("session: not bootstrapped")

// dogeolocate implements geolocate.
func (s *Session) dogeolocate(ctx context.Context, req *GeolocateRequest) (*geolocate.Results, error) {
	if s.state.IsNone() {
		return nil, ErrNotBootstrapped
	}

	ts := newTickerService(ctx, s)
	defer ts.stop()

	geolocateConfig := geolocate.Config{
		Resolver:  s.state.Unwrap().resolver,
		Logger:    s.state.Unwrap().logger,
		UserAgent: model.HTTPHeaderUserAgent, // do not disclose we are OONI
	}
	task := geolocate.NewTask(geolocateConfig) // TODO(bassosimone): make this a pointer.

	// TODO(bassosimone): we should make geolocate.Results a type
	// in the internal/model package and use the ~typical naming for
	// its fields rather than the naming we have here.
	location, err := task.Run(ctx)
	if err != nil {
		return nil, err
	}

	s.state.Unwrap().location = model.NewOptionalPtr(location)
	return location, nil
}
