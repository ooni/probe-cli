package engine

//
// ExperimentBuilder definition and implementation
//

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/registry"
)

// TODO(bassosimone,DecFox): we should eventually finish merging the code in
// file with the code inside the ./internal/registry package.
//
// If there's time, this could happen at the end of the current (as of 2024-06-27)
// richer input work, otherwise any time in the future is actually fine.

// experimentBuilder implements [model.ExperimentBuilder].
//
// This type is now just a tiny wrapper around registry.Factory.
type experimentBuilder struct {
	factory *registry.Factory

	// callbacks contains callbacks for the new experiment.
	callbacks model.ExperimentCallbacks

	// session is the session
	session *Session
}

var _ model.ExperimentBuilder = &experimentBuilder{}

// Interruptible implements [model.ExperimentBuilder].
func (b *experimentBuilder) Interruptible() bool {
	return b.factory.Interruptible()
}

// InputPolicy implements [model.ExperimentBuilder].
func (b *experimentBuilder) InputPolicy() model.InputPolicy {
	return b.factory.InputPolicy()
}

// Options implements [model.ExperimentBuilder].
func (b *experimentBuilder) Options() (map[string]model.ExperimentOptionInfo, error) {
	return b.factory.Options()
}

// SetOptionAny implements [model.ExperimentBuilder].
func (b *experimentBuilder) SetOptionAny(key string, value any) error {
	return b.factory.SetOptionAny(key, value)
}

// SetOptionsAny implements [model.ExperimentBuilder].
func (b *experimentBuilder) SetOptionsAny(options map[string]any) error {
	return b.factory.SetOptionsAny(options)
}

// SetOptionsJSON implements [model.ExperimentBuilder].
func (b *experimentBuilder) SetOptionsJSON(value json.RawMessage) error {
	return b.factory.SetOptionsJSON(value)
}

// SetCallbacks implements [model.ExperimentBuilder].
func (b *experimentBuilder) SetCallbacks(callbacks model.ExperimentCallbacks) {
	b.callbacks = callbacks
}

// NewExperiment creates a new [model.Experiment] instance.
func (b *experimentBuilder) NewExperiment() model.Experiment {
	measurer := b.factory.NewExperimentMeasurer()
	experiment := newExperiment(b.session, measurer)
	experiment.callbacks = b.callbacks
	return experiment
}

// NewTargetLoader creates a new [model.ExperimentTargetLoader] instance.
func (b *experimentBuilder) NewTargetLoader(config *model.ExperimentTargetLoaderConfig) model.ExperimentTargetLoader {
	return b.factory.NewTargetLoader(config)
}

// newExperimentBuilder creates a new [*experimentBuilder] instance.
func newExperimentBuilder(session *Session, name string) (*experimentBuilder, error) {
	factory, err := registry.NewFactory(name, session.kvStore, session.logger)
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
