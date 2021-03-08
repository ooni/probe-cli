package oonimkall

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// experimentSession is the abstract representation of
// a session according to an experiment.
type experimentSession interface {
	lock()
	maybeLookupBackendsContext(ctx context.Context) error
	maybeLookupLocationContext(ctx context.Context) error
	newExperimentBuilder(name string) (experimentBuilder, error)
	unlock()
}

// lock implements experimentSession.lock
func (sess *Session) lock() {
	sess.mtx.Lock()
}

// maybeLookupBackendsContext implements experimentSession.maybeLookupBackendsContext
func (sess *Session) maybeLookupBackendsContext(ctx context.Context) error {
	return sess.sessp.MaybeLookupBackendsContext(ctx)
}

// maybeLookupLocationContext implements experimentSession.maybeLookupLocationContext
func (sess *Session) maybeLookupLocationContext(ctx context.Context) error {
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

// experimentBuilderWrapper wraps *ExperimentBuilder
type experimentBuilderWrapper struct {
	eb *engine.ExperimentBuilder
}

func (eb *experimentBuilderWrapper) NewExperiment() experiment {
	return eb.eb.NewExperiment()
}

func (eb *experimentBuilderWrapper) SetCallbacks(cb ExperimentCallbacks) {
	eb.eb.SetCallbacks(cb)
}

// experimentBuilder is the representation of an experiment
// builder that we use inside this package.
type experimentBuilder interface {
	NewExperiment() experiment
	SetCallbacks(ExperimentCallbacks)
}

// experiment is the representation of an experiment that
// we use inside this package.
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
