package ooni

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/session"
)

// maybeBootstrap performs the session's bootstrap.
func (s *engineSession) maybeBootstrap() error {
	ctx := context.Background() // XXX
	if err := s.session.Send(ctx, &session.Request{Bootstrap: s.config}); err != nil {
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
		if resp.Ticker != nil {
			s.logger.Infof("bootstrap in progress (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.Bootstrap != nil {
			return resp.Bootstrap.Error
		}
		s.logger.Warnf("unexpected event: %+v", resp)
	}
}

// maybeLookupLocation geolocates the probe.
func (s *engineSession) maybeLookupLocation() (*geolocate.Results, error) {
	ctx := context.Background() // XXX
	if err := s.maybeBootstrap(); err != nil {
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
		if resp.Ticker != nil {
			s.logger.Infof("geolocate in progress (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.Geolocate != nil {
			return resp.Geolocate.Location, resp.Geolocate.Error
		}
		s.logger.Warnf("unexpected event: %+v", resp)
	}
}

// maybeCheckIn calls the check-in API.
func (s *engineSession) maybeCheckIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResultNettests, error) {
	if err := s.maybeBootstrap(); err != nil {
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
		if resp.Ticker != nil {
			s.logger.Infof("check-in in progress (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.CheckIn != nil {
			if resp.CheckIn.Error != nil {
				return nil, resp.CheckIn.Error
			}
			// While this code has been writting with single-goroutine usage
			// in mind, it seems safer to protect this variable anyway
			s.checkInMu.Lock()
			s.checkIn = model.NewOptionalPtr(resp.CheckIn.Result)
			s.checkInMu.Unlock()
			return &resp.CheckIn.Result.Tests, nil
		}
		s.logger.Warnf("unexpected event: %+v", resp)
	}
}

// emitLog emits a log event.
func (s *engineSession) emitLog(ev *session.LogEvent) {
	switch ev.Level {
	case "DEBUG":
		s.logger.Debug(ev.Message)
	case "WARNING":
		s.logger.Warn(ev.Message)
	default:
		s.logger.Info(ev.Message)
	}
}

// runWebConnectivity runs the Web Connectivity experiment.
func (me *modelExperiment) runWebConnectivity(
	ctx context.Context, input string) (*model.Measurement, error) {
	runtimex.Assert(me.measurer.ExperimentName() == "web_connectivity", "invalid experiment")
	req := &session.WebConnectivityRequest{
		Input:         input,
		ReportID:      me.meb.reportID,
		TestStartTime: me.testStartTime,
	}
	sess := me.meb.session.session
	if err := sess.Send(ctx, &session.Request{WebConnectivity: req}); err != nil {
		return nil, err
	}
	for {
		resp, err := sess.Recv(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Log != nil {
			me.meb.session.emitLog(resp.Log)
			continue
		}
		if resp.Ticker != nil {
			me.meb.session.logger.Infof("check-in in progress (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.WebConnectivity != nil {
			return resp.WebConnectivity.Measurement, resp.WebConnectivity.Error
		}
		me.meb.session.logger.Warnf("unexpected event: %+v", resp)
	}
}

// submit submites a measurement.
func (me *modelExperiment) submit(ctx context.Context, measurement *model.Measurement) error {
	sess := me.meb.session.session
	if err := sess.Send(ctx, &session.Request{Submit: measurement}); err != nil {
		return err
	}
	for {
		resp, err := sess.Recv(ctx)
		if err != nil {
			return err
		}
		if resp.Log != nil {
			me.meb.session.emitLog(resp.Log)
			continue
		}
		if resp.Ticker != nil {
			me.meb.session.logger.Infof("webconnectivity (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.Submit != nil {
			return resp.Submit.Error
		}
		me.meb.session.logger.Warnf("unexpected event: %+v", resp)
	}
}
