package oonimkall

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// FakeExperimentCallbacks contains fake ExperimentCallbacks.
type FakeExperimentCallbacks struct{}

// OnProgress implements ExperimentCallbacks.OnProgress
func (cb *FakeExperimentCallbacks) OnProgress(percentage float64, message string) {}

// FakeExperimentSession is a fake experimentSession
type FakeExperimentSession struct {
	ExperimentBuilder       experimentBuilder
	LockCount               *atomicx.Int64
	LookupBackendsErr       error
	LookupLocationErr       error
	NewExperimentBuilderErr error
	UnlockCount             *atomicx.Int64
}

// lock implements experimentSession.lock
func (sess *FakeExperimentSession) lock() {
	if sess.LockCount != nil {
		sess.LockCount.Add(1)
	}
}

// maybeLookupBackends implements experimentSession.maybeLookupBackends
func (sess *FakeExperimentSession) maybeLookupBackends(ctx context.Context) error {
	return sess.LookupBackendsErr
}

// maybeLookupLocation implements experimentSession.maybeLookupLocation
func (sess *FakeExperimentSession) maybeLookupLocation(ctx context.Context) error {
	return sess.LookupLocationErr
}

// newExperimentBuilder implements experimentSession.newExperimentBuilder
func (sess *FakeExperimentSession) newExperimentBuilder(name string) (experimentBuilder, error) {
	return sess.ExperimentBuilder, sess.NewExperimentBuilderErr
}

// unlock implements experimentSession.unlock
func (sess *FakeExperimentSession) unlock() {
	if sess.UnlockCount != nil {
		sess.UnlockCount.Add(1)
	}
}

// FakeExperimentBuilder is a fake experimentBuilder
type FakeExperimentBuilder struct {
	Callbacks  ExperimentCallbacks
	Experiment experiment
	mu         sync.Mutex
}

// newExperiment implements experimentBuilder.newExperiment
func (eb *FakeExperimentBuilder) newExperiment() experiment {
	return eb.Experiment
}

// setCallbacks implements experimentBuilder.setCallbacks
func (eb *FakeExperimentBuilder) setCallbacks(cb ExperimentCallbacks) {
	defer eb.mu.Unlock()
	eb.mu.Lock()
	eb.Callbacks = cb
}

// FakeExperiment is a fake experiment
type FakeExperiment struct {
	Err         error
	Measurement *model.Measurement
	Received    float64
	Sent        float64
}

// MeasureWithContext implements experiment.MeasureWithContext.
func (e *FakeExperiment) MeasureWithContext(ctx context.Context, input string) (
	measurement *model.Measurement, err error) {
	return e.Measurement, e.Err
}

// KibiBytesSent implements experiment.KibiBytesSent
func (e *FakeExperiment) KibiBytesSent() float64 {
	return e.Sent
}

// KibiBytesReceived implements experiment.KibiBytesReceived
func (e *FakeExperiment) KibiBytesReceived() float64 {
	return e.Received
}
