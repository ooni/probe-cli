package oonimkall

//
// This file contains mocks for types used by tasks. Because
// we only use mocks when testing, this file is a `_test.go` file.
//

import (
	"context"
	"errors"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// CollectorTaskEmitter is a thread-safe taskEmitter
// that stores all the events inside itself.
type CollectorTaskEmitter struct {
	// events contains the events
	events []*event

	// mu provides mutual exclusion
	mu sync.Mutex
}

// ensures that a CollectorTaskEmitter is a taskEmitter.
var _ taskEmitter = &CollectorTaskEmitter{}

// Emit implements the taskEmitter.Emit method.
func (e *CollectorTaskEmitter) Emit(key string, value interface{}) {
	e.mu.Lock()
	e.events = append(e.events, &event{Key: key, Value: value})
	e.mu.Unlock()
}

// Collect returns a copy of the collected events. It is safe
// to read the events. It's a data race to modify them.
//
// After this function has been called, the internal array
// of events will now be empty.
func (e *CollectorTaskEmitter) Collect() (out []*event) {
	e.mu.Lock()
	out = e.events
	e.events = nil
	e.mu.Unlock()
	return
}

// SessionBuilderConfigSaver is a session builder that
// saves the received config and returns an error.
type SessionBuilderConfigSaver struct {
	Config engine.SessionConfig
}

var _ taskSessionBuilder = &SessionBuilderConfigSaver{}

func (b *SessionBuilderConfigSaver) NewSession(
	ctx context.Context, config engine.SessionConfig) (taskSession, error) {
	b.Config = config
	return nil, errors.New("mocked error")
}

// MockableTaskRunnerDependencies allows to mock all the
// dependencies of taskRunner using a single structure.
type MockableTaskRunnerDependencies struct {

	// taskSessionBuilder:

	MockNewSession func(ctx context.Context,
		config engine.SessionConfig) (taskSession, error)

	// taskSession:

	MockClose                      func() error
	MockNewExperimentBuilder       func(name string) (model.ExperimentBuilder, error)
	MockMaybeLookupBackendsContext func(ctx context.Context) error
	MockMaybeLookupLocationContext func(ctx context.Context) error
	MockProbeIP                    func() string
	MockProbeASNString             func() string
	MockProbeCC                    func() string
	MockProbeNetworkName           func() string
	MockResolverASNString          func() string
	MockResolverIP                 func() string
	MockResolverNetworkName        func() string

	// model.ExperimentBuilder:

	MockableSetCallbacks  func(callbacks model.ExperimentCallbacks)
	MockableInputPolicy   func() model.InputPolicy
	MockableNewExperiment func() model.Experiment
	MockableInterruptible func() bool

	// model.Experiment:

	MockableKibiBytesReceived  func() float64
	MockableKibiBytesSent      func() float64
	MockableOpenReportContext  func(ctx context.Context) error
	MockableReportID           func() string
	MockableMeasureWithContext func(ctx context.Context, input string) (
		measurement *model.Measurement, err error)
	MockableSubmitAndUpdateMeasurementContext func(
		ctx context.Context, measurement *model.Measurement) error
}

var (
	_ taskSessionBuilder      = &MockableTaskRunnerDependencies{}
	_ taskSession             = &MockableTaskRunnerDependencies{}
	_ model.ExperimentBuilder = &MockableTaskRunnerDependencies{}
	_ model.Experiment        = &MockableTaskRunnerDependencies{}
)

// NewSession implements taskSessionBuilder
func (dep *MockableTaskRunnerDependencies) NewSession(
	ctx context.Context, config engine.SessionConfig) (taskSession, error) {
	if f := dep.MockNewSession; f != nil {
		return f(ctx, config)
	}
	return dep, nil
}

// Close implements taskSession
func (dep *MockableTaskRunnerDependencies) Close() error {
	return dep.MockClose()
}

// NewExperimentBuilder implements taskSession
func (dep *MockableTaskRunnerDependencies) NewExperimentBuilder(name string) (model.ExperimentBuilder, error) {
	if f := dep.MockNewExperimentBuilder; f != nil {
		return f(name)
	}
	return dep, nil
}

// MaybeLookupBackendsContext implements taskSession
func (dep *MockableTaskRunnerDependencies) MaybeLookupBackendsContext(ctx context.Context) error {
	return dep.MockMaybeLookupBackendsContext(ctx)
}

// MaybeLookupLocationContext implements taskSession
func (dep *MockableTaskRunnerDependencies) MaybeLookupLocationContext(ctx context.Context) error {
	return dep.MockMaybeLookupLocationContext(ctx)
}

// ProbeIP implements taskSession
func (dep *MockableTaskRunnerDependencies) ProbeIP() string {
	return dep.MockProbeIP()
}

// ProbeASNString implements taskSession
func (dep *MockableTaskRunnerDependencies) ProbeASNString() string {
	return dep.MockProbeASNString()
}

// ProbeCC implements taskSession
func (dep *MockableTaskRunnerDependencies) ProbeCC() string {
	return dep.MockProbeCC()
}

// ProbeNetworkName implements taskSession
func (dep *MockableTaskRunnerDependencies) ProbeNetworkName() string {
	return dep.MockProbeNetworkName()
}

// ResolverASNString implements taskSession
func (dep *MockableTaskRunnerDependencies) ResolverASNString() string {
	return dep.MockResolverASNString()
}

// ResolverIP implements taskSession
func (dep *MockableTaskRunnerDependencies) ResolverIP() string {
	return dep.MockResolverIP()
}

// ResolverNetworkName implements taskSession
func (dep *MockableTaskRunnerDependencies) ResolverNetworkName() string {
	return dep.MockResolverNetworkName()
}

// BuildRicherInput implements model.ExperimentBuilder.
func (dep *MockableTaskRunnerDependencies) BuildRicherInput(annotations map[string]string, flatInputs []string) []model.RicherInput {
	// This method is unimplemented because it's not used by oonimkall
	panic("unimplemented")
}

// NewRicherInputExperiment implements model.ExperimentBuilder.
func (dep *MockableTaskRunnerDependencies) NewRicherInputExperiment() model.RicherInputExperiment {
	// This method is unimplemented because it's not used by oonimkall
	panic("unimplemented")
}

// Options implements model.ExperimentBuilder.
func (dep *MockableTaskRunnerDependencies) Options() (map[string]model.ExperimentOptionInfo, error) {
	// This method is unimplemented because it's not used by oonimkall
	panic("unimplemented")
}

// SetOptionAny implements model.ExperimentBuilder.
func (dep *MockableTaskRunnerDependencies) SetOptionAny(key string, value any) error {
	// This method is unimplemented because it's not used by oonimkall
	panic("unimplemented")
}

// SetOptionsAny implements model.ExperimentBuilder.
func (dep *MockableTaskRunnerDependencies) SetOptionsAny(options map[string]any) error {
	// This method is unimplemented because it's not used by oonimkall
	panic("unimplemented")
}

// SetCallbacks implements model.ExperimentBuilder
func (dep *MockableTaskRunnerDependencies) SetCallbacks(callbacks model.ExperimentCallbacks) {
	dep.MockableSetCallbacks(callbacks)
}

// InputPolicy implements model.ExperimentBuilder
func (dep *MockableTaskRunnerDependencies) InputPolicy() model.InputPolicy {
	return dep.MockableInputPolicy()
}

// NewExperiment implements model.ExperimentBuilder
func (dep *MockableTaskRunnerDependencies) NewExperiment() model.Experiment {
	if f := dep.MockableNewExperiment; f != nil {
		return f()
	}
	return dep
}

// Interruptible implements model.ExperimentBuilder
func (dep *MockableTaskRunnerDependencies) Interruptible() bool {
	return dep.MockableInterruptible()
}

// KibiBytesReceived implements model.Experiment
func (dep *MockableTaskRunnerDependencies) KibiBytesReceived() float64 {
	return dep.MockableKibiBytesReceived()
}

// KibiBytesSent implements model.Experiment
func (dep *MockableTaskRunnerDependencies) KibiBytesSent() float64 {
	return dep.MockableKibiBytesSent()
}

// OpenReportContext implements model.Experiment
func (dep *MockableTaskRunnerDependencies) OpenReportContext(ctx context.Context) error {
	return dep.MockableOpenReportContext(ctx)
}

// ReportID implements model.Experiment
func (dep *MockableTaskRunnerDependencies) ReportID() string {
	return dep.MockableReportID()
}

// MeasureAsync implements model.Experiment.
func (dep *MockableTaskRunnerDependencies) MeasureAsync(ctx context.Context, input string) (<-chan *model.Measurement, error) {
	// This method is unimplemented because it's not used by oonimkall
	panic("unimplemented")
}

// MeasureWithContext implements model.Experiment
func (dep *MockableTaskRunnerDependencies) MeasureWithContext(ctx context.Context, input string) (
	measurement *model.Measurement, err error) {
	return dep.MockableMeasureWithContext(ctx, input)
}

// Name implements model.Experiment.
func (dep *MockableTaskRunnerDependencies) Name() string {
	// This method is unimplemented because it's not used by oonimkall
	panic("unimplemented")
}

// SubmitAndUpdateMeasurementContext implements model.Experiment
func (dep *MockableTaskRunnerDependencies) SubmitAndUpdateMeasurementContext(
	ctx context.Context, measurement *model.Measurement) error {
	return dep.MockableSubmitAndUpdateMeasurementContext(ctx, measurement)
}

// MockableKVStoreFSBuilder is a mockable taskKVStoreFSBuilder.
type MockableKVStoreFSBuilder struct {
	MockNewFS func(path string) (model.KeyValueStore, error)
}

var _ taskKVStoreFSBuilder = &MockableKVStoreFSBuilder{}

func (m *MockableKVStoreFSBuilder) NewFS(path string) (model.KeyValueStore, error) {
	return m.MockNewFS(path)
}
