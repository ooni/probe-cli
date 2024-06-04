package mocks

import "github.com/ooni/probe-cli/v3/internal/model"

// ExperimentBuilder mocks model.ExperimentBuilder.
type ExperimentBuilder struct {
	MockInterruptible func() bool

	MockInputPolicy func() model.InputPolicy

	MockOptions func() (map[string]model.ExperimentOptionInfo, error)

	MockSetOptionAny func(key string, value any) error

	MockSetOptionsAny func(options map[string]any) error

	MockSetCallbacks func(callbacks model.ExperimentCallbacks)

	MockNewExperiment func() model.Experiment
}

func (eb *ExperimentBuilder) Interruptible() bool {
	return eb.MockInterruptible()
}

func (eb *ExperimentBuilder) InputPolicy() model.InputPolicy {
	return eb.MockInputPolicy()
}

func (eb *ExperimentBuilder) Options() (map[string]model.ExperimentOptionInfo, error) {
	return eb.MockOptions()
}

func (eb *ExperimentBuilder) SetOptionAny(key string, value any) error {
	return eb.MockSetOptionAny(key, value)
}

func (eb *ExperimentBuilder) SetOptionsAny(options map[string]any) error {
	return eb.MockSetOptionsAny(options)
}

func (eb *ExperimentBuilder) SetCallbacks(callbacks model.ExperimentCallbacks) {
	eb.MockSetCallbacks(callbacks)
}

func (eb *ExperimentBuilder) NewExperiment() model.Experiment {
	return eb.MockNewExperiment()
}
