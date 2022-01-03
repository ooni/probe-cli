package oonimkall

import (
	"context"
	"errors"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
)

//
// This file contains mocks for types used by tasks. Because
// we only use mocks when testing, this file is a `_test.go` file.
//

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
	MockNewExperimentBuilderByName func(name string) (taskExperimentBuilder, error)
	MockMaybeLookupBackendsContext func(ctx context.Context) error
	MockMaybeLookupLocationContext func(ctx context.Context) error
	MockProbeIP                    func() string
	MockProbeASNString             func() string
	MockProbeCC                    func() string
	MockProbeNetworkName           func() string
	MockResolverASNString          func() string
	MockResolverIP                 func() string
	MockResolverNetworkName        func() string

	// taskExperimentBuilder:

	MockableSetCallbacks          func(callbacks model.ExperimentCallbacks)
	MockableInputPolicy           func() engine.InputPolicy
	MockableNewExperimentInstance func() taskExperiment
	MockableInterruptible         func() bool

	// taskExperiment:

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
	_ taskSessionBuilder    = &MockableTaskRunnerDependencies{}
	_ taskSession           = &MockableTaskRunnerDependencies{}
	_ taskExperimentBuilder = &MockableTaskRunnerDependencies{}
	_ taskExperiment        = &MockableTaskRunnerDependencies{}
)

func (dep *MockableTaskRunnerDependencies) NewSession(
	ctx context.Context, config engine.SessionConfig) (taskSession, error) {
	if f := dep.MockNewSession; f != nil {
		return f(ctx, config)
	}
	return dep, nil
}

func (dep *MockableTaskRunnerDependencies) Close() error {
	return dep.MockClose()
}

func (dep *MockableTaskRunnerDependencies) NewExperimentBuilderByName(name string) (taskExperimentBuilder, error) {
	if f := dep.MockNewExperimentBuilderByName; f != nil {
		return f(name)
	}
	return dep, nil
}

func (dep *MockableTaskRunnerDependencies) MaybeLookupBackendsContext(ctx context.Context) error {
	return dep.MockMaybeLookupBackendsContext(ctx)
}

func (dep *MockableTaskRunnerDependencies) MaybeLookupLocationContext(ctx context.Context) error {
	return dep.MockMaybeLookupLocationContext(ctx)
}

func (dep *MockableTaskRunnerDependencies) ProbeIP() string {
	return dep.MockProbeIP()
}

func (dep *MockableTaskRunnerDependencies) ProbeASNString() string {
	return dep.MockProbeASNString()
}

func (dep *MockableTaskRunnerDependencies) ProbeCC() string {
	return dep.MockProbeCC()
}

func (dep *MockableTaskRunnerDependencies) ProbeNetworkName() string {
	return dep.MockProbeNetworkName()
}

func (dep *MockableTaskRunnerDependencies) ResolverASNString() string {
	return dep.MockResolverASNString()
}

func (dep *MockableTaskRunnerDependencies) ResolverIP() string {
	return dep.MockResolverIP()
}

func (dep *MockableTaskRunnerDependencies) ResolverNetworkName() string {
	return dep.MockResolverNetworkName()
}

func (dep *MockableTaskRunnerDependencies) SetCallbacks(callbacks model.ExperimentCallbacks) {
	dep.MockableSetCallbacks(callbacks)
}

func (dep *MockableTaskRunnerDependencies) InputPolicy() engine.InputPolicy {
	return dep.MockableInputPolicy()
}

func (dep *MockableTaskRunnerDependencies) NewExperimentInstance() taskExperiment {
	if f := dep.MockableNewExperimentInstance; f != nil {
		return f()
	}
	return dep
}

func (dep *MockableTaskRunnerDependencies) Interruptible() bool {
	return dep.MockableInterruptible()
}

func (dep *MockableTaskRunnerDependencies) KibiBytesReceived() float64 {
	return dep.MockableKibiBytesReceived()
}

func (dep *MockableTaskRunnerDependencies) KibiBytesSent() float64 {
	return dep.MockableKibiBytesSent()
}

func (dep *MockableTaskRunnerDependencies) OpenReportContext(ctx context.Context) error {
	return dep.MockableOpenReportContext(ctx)
}

func (dep *MockableTaskRunnerDependencies) ReportID() string {
	return dep.MockableReportID()
}

func (dep *MockableTaskRunnerDependencies) MeasureWithContext(ctx context.Context, input string) (
	measurement *model.Measurement, err error) {
	return dep.MockableMeasureWithContext(ctx, input)
}

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
