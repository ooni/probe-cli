package ooni

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/registry"
	"github.com/ooni/probe-cli/v3/internal/session"
)

// engineSession emulates [engine.engineSession] using [session.engineSession].
type engineSession struct {
	// checkIn stores the most recent check-in API response (if any).
	checkIn model.OptionalPtr[model.OOAPICheckInResult]

	// checkInMu protects the checkIn field
	checkInMu sync.Mutex

	// config contains the config for bootstrapping the session.
	config *session.BootstrapRequest

	// logger is the logger to use.
	logger model.Logger

	// once allows us to run cleanups just once.
	once sync.Once

	// session is the initially empty session.
	session *session.Session
}

var (
	_ ProbeEngine               = &engineSession{}
	_ engine.InputLoaderSession = &engineSession{}
	_ model.LocationProvider    = &engineSession{}
)

// newSession creates a new instance of Session.
func newSession(config *session.BootstrapRequest, logger model.Logger) *engineSession {
	return &engineSession{
		checkIn:   model.OptionalPtr[model.OOAPICheckInResult]{},
		checkInMu: sync.Mutex{},
		config:    config,
		logger:    logger,
		once:      sync.Once{},
		session:   session.New(),
	}
}

// SoftwareName implements ProbeEngine
func (s *engineSession) SoftwareName() string {
	return s.config.SoftwareName
}

// SoftwareVersion implements ProbeEngine
func (s *engineSession) SoftwareVersion() string {
	return s.config.SoftwareVersion
}

// Close implements ProbeEngine
func (s *engineSession) Close() error {
	s.once.Do(func() {
		s.session.Close()
	})
	return nil
}

// MaybeLookupLocation implements ProbeEngine
func (s *engineSession) MaybeLookupLocation() error {
	if _, err := s.maybeLookupLocation(); err != nil {
		return err
	}
	return nil
}

// ProbeASNString implements ProbeEngine
func (s *engineSession) ProbeASNString() string {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultProbeASNString
	}
	return location.ProbeASNString()
}

// ProbeCC implements ProbeEngine
func (s *engineSession) ProbeCC() string {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultProbeCC
	}
	return location.ProbeCC()
}

// ProbeIP implements ProbeEngine
func (s *engineSession) ProbeIP() string {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultProbeIP
	}
	return location.ProbeIP()
}

// ProbeNetworkName implements ProbeEngine
func (s *engineSession) ProbeNetworkName() string {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultProbeNetworkName
	}
	return location.ProbeNetworkName()
}

// ProbeASN implements model.LocationProvider
func (s *engineSession) ProbeASN() uint {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultProbeASN
	}
	return location.ProbeASN()
}

// ResolverASN implements model.LocationProvider
func (s *engineSession) ResolverASN() uint {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultResolverASN
	}
	return location.ResolverASN()
}

// ResolverASNString implements model.LocationProvider
func (s *engineSession) ResolverASNString() string {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultResolverASNString
	}
	return location.ResolverASNString()
}

// ResolverIP implements model.LocationProvider
func (s *engineSession) ResolverIP() string {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultResolverIP
	}
	return location.ResolverIP()
}

// ResolverNetworkName implements model.LocationProvider
func (s *engineSession) ResolverNetworkName() string {
	location, err := s.maybeLookupLocation()
	if err != nil {
		return model.DefaultResolverNetworkName
	}
	return location.ResolverNetworkName()
}

// MaybeLookupBackends implements ProbeEngine
func (s *engineSession) MaybeLookupBackends(config *model.OOAPICheckInConfig) error {
	_, err := s.maybeCheckIn(context.Background(), config)
	return err
}

// CheckIn implements engine.InputLoaderSession
func (s *engineSession) CheckIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResultNettests, error) {
	return s.maybeCheckIn(ctx, config)
}

// NewExperimentBuilder implements ProbeEngine
func (s *engineSession) NewExperimentBuilder(name string) (model.ExperimentBuilder, error) {
	factory, err := registry.NewFactory(name)
	if err != nil {
		return nil, err
	}

	// Lock because we are accessing the cached check-in
	defer s.checkInMu.Unlock()
	s.checkInMu.Lock()
	if s.checkIn.IsNone() {
		return nil, errors.New("no cached check-in API response")
	}
	resp := s.checkIn.Unwrap()

	var reportID string
	switch name {
	case "web_connectivity":
		if resp.Tests.WebConnectivity == nil {
			return nil, errors.New("no experiment-specific info in check-in API response")
		}
		reportID = resp.Tests.WebConnectivity.ReportID
	default:
		return nil, errors.New("not implemented")
	}

	meb := &modelExperimentBuilder{
		callbacks: nil,
		factory:   factory,
		reportID:  reportID,
		session:   s,
	}
	return meb, nil
}

// modelExperimentBuilder implements [model.ExperimentBuilder] using [session.Session].
type modelExperimentBuilder struct {
	callbacks model.ExperimentCallbacks
	factory   *registry.Factory
	reportID  string
	session   *engineSession
}

var _ model.ExperimentBuilder = &modelExperimentBuilder{}

// InputPolicy implements model.ExperimentBuilder
func (meb *modelExperimentBuilder) InputPolicy() model.InputPolicy {
	return meb.factory.InputPolicy()
}

// Interruptible implements model.ExperimentBuilder
func (meb *modelExperimentBuilder) Interruptible() bool {
	return meb.factory.Interruptible()
}

// NewExperiment implements model.ExperimentBuilder
func (meb *modelExperimentBuilder) NewExperiment() model.Experiment {
	measurer := meb.factory.NewExperimentMeasurer()
	me := &modelExperiment{
		bc:            bytecounter.New(),
		measurer:      measurer,
		meb:           meb,
		testStartTime: time.Now(),
	}
	return me
}

// Options implements model.ExperimentBuilder
func (meb *modelExperimentBuilder) Options() (map[string]model.ExperimentOptionInfo, error) {
	return meb.factory.Options()
}

// SetCallbacks implements model.ExperimentBuilder
func (meb *modelExperimentBuilder) SetCallbacks(callbacks model.ExperimentCallbacks) {
	meb.callbacks = callbacks
}

// SetOptionAny implements model.ExperimentBuilder
func (meb *modelExperimentBuilder) SetOptionAny(key string, value any) error {
	return meb.factory.SetOptionAny(key, value)
}

// SetOptionsAny implements model.ExperimentBuilder
func (meb *modelExperimentBuilder) SetOptionsAny(options map[string]any) error {
	return meb.factory.SetOptionsAny(options)
}

// modelExperiment implements [model.Experiment] using [session.Session].
type modelExperiment struct {
	bc            *bytecounter.Counter
	measurer      model.ExperimentMeasurer
	meb           *modelExperimentBuilder
	testStartTime time.Time
}

var _ model.Experiment = &modelExperiment{}

// GetSummaryKeys implements model.Experiment
func (me *modelExperiment) GetSummaryKeys(m *model.Measurement) (any, error) {
	return me.measurer.GetSummaryKeys(m)
}

// KibiBytesReceived implements model.Experiment
func (me *modelExperiment) KibiBytesReceived() float64 {
	return me.bc.KibiBytesReceived()
}

// KibiBytesSent implements model.Experiment
func (me *modelExperiment) KibiBytesSent() float64 {
	return me.bc.KibiBytesSent()
}

// MeasureAsync implements model.Experiment
func (me *modelExperiment) MeasureAsync(ctx context.Context, input string) (<-chan *model.Measurement, error) {
	return nil, errors.New("not implemented")
}

// MeasureWithContext implements model.Experiment
func (me *modelExperiment) MeasureWithContext(ctx context.Context, input string) (*model.Measurement, error) {
	// XXX: bytes sent and received?
	switch me.measurer.ExperimentName() {
	case "web_connectivity":
		return me.runWebConnectivity(ctx, input)
	default:
		return nil, errors.New("not implemented")
	}
}

// Name implements model.Experiment
func (me *modelExperiment) Name() string {
	return me.measurer.ExperimentName()
}

// OpenReportContext implements model.Experiment
func (me *modelExperiment) OpenReportContext(ctx context.Context) error {
	return nil
}

// ReportID implements model.Experiment
func (me *modelExperiment) ReportID() string {
	return me.meb.reportID
}

// SaveMeasurement implements model.Experiment
func (me *modelExperiment) SaveMeasurement(measurement *model.Measurement, filePath string) error {
	data, err := json.Marshal(measurement)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

// SubmitAndUpdateMeasurementContext implements model.Experiment
func (me *modelExperiment) SubmitAndUpdateMeasurementContext(ctx context.Context, measurement *model.Measurement) error {
	// Note: the measurement has already the correct reportID since the beginning
	return me.submit(ctx, measurement)
}
