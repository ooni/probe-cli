package oonimkall

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// experimentSession is the abstract representation of
// a session according to an experiment.
type experimentSession interface {
	// lock locks the session
	lock()

	// maybeLookupBackends lookups the backends
	maybeLookupBackends(ctx context.Context) error

	// maybeLookupLocations lookups the probe location
	maybeLookupLocation(ctx context.Context) error

	// newExperimentBuilder creates a new experiment builder
	newExperimentBuilder(name string) (experimentBuilder, error)

	// unlock unlocks the session
	unlock()
}

// lock implements experimentSession.lock
func (sess *Session) lock() {
	sess.mtx.Lock()
}

// maybeLookupBackends implements experimentSession.maybeLookupBackends
func (sess *Session) maybeLookupBackends(ctx context.Context) error {
	return sess.sessp.MaybeLookupBackendsContext(ctx)
}

// maybeLookupLocation implements experimentSession.maybeLookupLocation
func (sess *Session) maybeLookupLocation(ctx context.Context) error {
	return sess.sessp.MaybeLookupLocationContext(ctx)
}

// newExperimentBuilder implements experimentSession.newExperimentBuilder
func (sess *Session) newExperimentBuilder(name string) (experimentBuilder, error) {
	eb, err := sess.sessp.NewExperimentBuilder(name)
	if err != nil {
		return nil, err
	}
	return &experimentBuilderWrapper{eb: eb}, nil
}

// unlock implements experimentSession.unlock
func (sess *Session) unlock() {
	sess.mtx.Unlock()
}

// experimentBuilder is the representation of an experiment
// builder that we use inside this package.
type experimentBuilder interface {
	// newExperiment creates a new experiment instance
	newExperiment() experiment

	// setCallbacks sets the experiment callbacks
	setCallbacks(ExperimentCallbacks)
}

// experimentBuilderWrapper wraps *ExperimentBuilder
type experimentBuilderWrapper struct {
	eb model.ExperimentBuilder
}

// newExperiment implements experimentBuilder.newExperiment
func (eb *experimentBuilderWrapper) newExperiment() experiment {
	return eb.eb.NewExperiment()
}

// setCallbacks implements experimentBuilder.setCallbacks
func (eb *experimentBuilderWrapper) setCallbacks(cb ExperimentCallbacks) {
	eb.eb.SetCallbacks(cb)
}

// experiment is the representation of an experiment that
// we use inside this package for running nettests.
type experiment interface {
	// MeasureWithContext runs the measurement with the given input
	// and context. It returns a measurement or an error.
	MeasureWithContext(ctx context.Context, input string) (
		measurement *model.Measurement, err error)

	// KibiBytesSent returns the number of KiB sent.
	KibiBytesSent() float64

	// KibiBytesReceived returns the number of KiB received.
	KibiBytesReceived() float64
}
