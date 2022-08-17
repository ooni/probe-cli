package engine

//
// ExperimentBuilder definition and implementation
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/registry"
)

// experimentBuilder implements ExperimentBuilder.
//
// This type is now just a tiny wrapper around registry.Factory.
type experimentBuilder struct {
	factory *registry.Factory

	// callbacks contains callbacks for the new experiment.
	callbacks model.ExperimentCallbacks

	// session is the session
	session *Session
}

// Interruptible implements ExperimentBuilder.Interruptible.
func (b *experimentBuilder) Interruptible() bool {
	return b.factory.Interruptible()
}

// InputPolicy implements ExperimentBuilder.InputPolicy.
func (b *experimentBuilder) InputPolicy() model.InputPolicy {
	return b.factory.InputPolicy()
}

// Options implements ExperimentBuilder.Options.
func (b *experimentBuilder) Options() (map[string]model.ExperimentOptionInfo, error) {
	return b.factory.Options()
}

// SetOptionAny implements ExperimentBuilder.SetOptionAny.
func (b *experimentBuilder) SetOptionAny(key string, value any) error {
	return b.factory.SetOptionAny(key, value)
}

// SetOptionsAny implements ExperimentBuilder.SetOptionsAny.
func (b *experimentBuilder) SetOptionsAny(options map[string]any) error {
	return b.factory.SetOptionsAny(options)
}

// SetCallbacks implements ExperimentBuilder.SetCallbacks.
func (b *experimentBuilder) SetCallbacks(callbacks model.ExperimentCallbacks) {
	b.callbacks = callbacks
}

// NewExperiment creates the experiment
func (b *experimentBuilder) NewExperiment() model.Experiment {
	measurer := b.factory.NewExperimentMeasurer()
	experiment := newExperiment(b.session, measurer)
	experiment.callbacks = b.callbacks
	return experiment
}

// newExperimentBuilder creates a new experimentBuilder instance.
func newExperimentBuilder(session *Session, name string) (*experimentBuilder, error) {
	factory, err := registry.NewFactory(name)
	if err != nil {
		return nil, err
	}
	builder := &experimentBuilder{
		factory:   factory,
		callbacks: model.NewPrinterCallbacks(session.Logger()),
		session:   session,
	}
	return builder, nil
}
