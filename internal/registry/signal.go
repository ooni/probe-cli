package registry

//
// Registers the `signal' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "signal"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return signal.NewExperimentMeasurer(
					*config.(*signal.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &signal.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		}
	}
}
