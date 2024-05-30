package registry

//
// Registers the `signal' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["signal"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
				return signal.NewExperimentMeasurer(
					*config.(*signal.Config),
				)
			},
			buildRicherInputExperiment: signal.NewRicherInputExperiment,
			config:                     &signal.Config{},
			enabledByDefault:           true,
			inputPolicy:                model.InputNone,
		}
	}
}
