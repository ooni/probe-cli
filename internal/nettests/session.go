package nettests

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/session"
	"golang.org/x/net/context"
)

// Session is a measurement session.
type Session struct {
	// bootstrapRequest contains settings to bootstrap a session.
	bootstrapRequest *session.BootstrapRequest

	// logger is the logger to use.
	logger model.Logger

	// once allows us to run cleanups just once.
	once sync.Once

	// session is the initially empty session.
	session *session.Session
}

// NewSession creates a new [Session] instance.
func NewSession(request *session.BootstrapRequest, logger model.Logger) *Session {
	return &Session{
		bootstrapRequest: request,
		logger:           logger,
		once:             sync.Once{},
		session:          session.New(),
	}
}

// Close implements ProbeEngine
func (s *Session) Close() error {
	s.once.Do(func() {
		s.session.Close()
	})
	return nil
}

// Bootstrap performs the session bootstrap.
func (s *Session) Bootstrap(ctx context.Context) error {
	if err := s.session.Send(ctx, &session.Request{Bootstrap: s.bootstrapRequest}); err != nil {
		return err
	}
	for {
		resp, err := s.session.Recv(ctx)
		if err != nil {
			return err
		}
		if resp.Log != nil {
			s.emitLog(resp.Log)
			continue
		}
		if resp.Bootstrap != nil {
			return resp.Bootstrap.Error
		}
	}
}

// Geolocate runs the geolocate task.
func (s *Session) Geolocate(ctx context.Context) (*geolocate.Results, error) {
	if err := s.Bootstrap(ctx); err != nil {
		return nil, err
	}
	req := &session.GeolocateRequest{}
	if err := s.session.Send(ctx, &session.Request{Geolocate: req}); err != nil {
		return nil, err
	}
	for {
		resp, err := s.session.Recv(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Log != nil {
			s.emitLog(resp.Log)
			continue
		}
		if resp.Geolocate != nil {
			return resp.Geolocate.Location, resp.Geolocate.Error
		}
	}
}

// CheckIn runs the checkIn task.
func (s *Session) CheckIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	if err := s.Bootstrap(ctx); err != nil {
		return nil, err
	}
	if err := s.session.Send(ctx, &session.Request{CheckIn: config}); err != nil {
		return nil, err
	}
	for {
		resp, err := s.session.Recv(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Log != nil {
			s.emitLog(resp.Log)
			continue
		}
		if resp.CheckIn != nil {
			return resp.CheckIn.Result, resp.CheckIn.Error
		}
	}
}

// Submit submits the given measurement.
func (s *Session) Submit(ctx context.Context, measurement *model.Measurement) error {
	if err := s.session.Send(ctx, &session.Request{Submit: measurement}); err != nil {
		return err
	}
	for {
		resp, err := s.session.Recv(ctx)
		if err != nil {
			return err
		}
		if resp.Log != nil {
			s.emitLog(resp.Log)
			continue
		}
		if resp.Submit != nil {
			return resp.Submit.Error
		}
	}
}

// emitLog emits a log event.
func (s *Session) emitLog(ev *session.LogEvent) {
	switch ev.Level {
	case "DEBUG":
		s.logger.Debug(ev.Message)
	case "WARNING":
		s.logger.Warn(ev.Message)
	default:
		s.logger.Info(ev.Message)
	}
}
