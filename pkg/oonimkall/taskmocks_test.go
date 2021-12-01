package oonimkall

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
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

// MockableSessionBuilder is a mockable taskSessionBuilder.
type MockableSessionBuilder struct {
	MockNewSession func(ctx context.Context,
		config engine.SessionConfig) (taskSession, error)
}

var _ taskSessionBuilder = &MockableSessionBuilder{}

func (b *MockableSessionBuilder) NewSession(
	ctx context.Context, config engine.SessionConfig) (taskSession, error) {
	return b.MockNewSession(ctx, config)
}

// MockableSession is a mockable taskSession.
type MockableSession struct {
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
}

var _ taskSession = &MockableSession{}

func (sess *MockableSession) Close() error {
	return sess.MockClose()
}

func (sess *MockableSession) NewExperimentBuilderByName(name string) (taskExperimentBuilder, error) {
	return sess.MockNewExperimentBuilderByName(name)
}

func (sess *MockableSession) MaybeLookupBackendsContext(ctx context.Context) error {
	return sess.MockMaybeLookupBackendsContext(ctx)
}

func (sess *MockableSession) MaybeLookupLocationContext(ctx context.Context) error {
	return sess.MockMaybeLookupLocationContext(ctx)
}

func (sess *MockableSession) ProbeIP() string {
	return sess.MockProbeIP()
}

func (sess *MockableSession) ProbeASNString() string {
	return sess.MockProbeASNString()
}

func (sess *MockableSession) ProbeCC() string {
	return sess.MockProbeCC()
}

func (sess *MockableSession) ProbeNetworkName() string {
	return sess.MockProbeNetworkName()
}

func (sess *MockableSession) ResolverASNString() string {
	return sess.MockResolverASNString()
}

func (sess *MockableSession) ResolverIP() string {
	return sess.MockResolverIP()
}

func (sess *MockableSession) ResolverNetworkName() string {
	return sess.MockResolverNetworkName()
}

// MockableExperimentBuilder is a mockable taskExperimentBuilder.
type MockableExperimentBuilder struct {
	MockableSetCallbacks          func(callbacks model.ExperimentCallbacks)
	MockableInputPolicy           func() engine.InputPolicy
	MockableNewExperimentInstance func() taskExperiment
	MockableInterruptible         func() bool
}

var _ taskExperimentBuilder = &MockableExperimentBuilder{}

func (b *MockableExperimentBuilder) SetCallbacks(callbacks model.ExperimentCallbacks) {
	b.MockableSetCallbacks(callbacks)
}

func (b *MockableExperimentBuilder) InputPolicy() engine.InputPolicy {
	return b.MockableInputPolicy()
}

func (b *MockableExperimentBuilder) NewExperimentInstance() taskExperiment {
	return b.MockableNewExperimentInstance()
}

func (b *MockableExperimentBuilder) Interruptible() bool {
	return b.MockableInterruptible()
}

// MockableExperiment is a mockable taskExperiment.
type MockableExperiment struct {
	MockableKibiBytesReceived func() float64

	MockableKibiBytesSent func() float64

	MockableOpenReportContext func(ctx context.Context) error

	MockableReportID func() string

	MockableMeasureWithContext func(ctx context.Context, input string) (
		measurement *model.Measurement, err error)

	MockableSubmitAndUpdateMeasurementContext func(
		ctx context.Context, measurement *model.Measurement) error
}

var _ taskExperiment = &MockableExperiment{}

func (exp *MockableExperiment) KibiBytesReceived() float64 {
	return exp.MockableKibiBytesReceived()
}

func (exp *MockableExperiment) KibiBytesSent() float64 {
	return exp.MockableKibiBytesSent()
}

func (exp *MockableExperiment) OpenReportContext(ctx context.Context) error {
	return exp.MockableOpenReportContext(ctx)
}

func (exp *MockableExperiment) ReportID() string {
	return exp.MockableReportID()
}

func (exp *MockableExperiment) MeasureWithContext(ctx context.Context, input string) (
	measurement *model.Measurement, err error) {
	return exp.MockableMeasureWithContext(ctx, input)
}

func (exp *MockableExperiment) SubmitAndUpdateMeasurementContext(
	ctx context.Context, measurement *model.Measurement) error {
	return exp.MockableSubmitAndUpdateMeasurementContext(ctx, measurement)
}
