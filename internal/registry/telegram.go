package registry

//
// Registers the `telegram' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["telegram"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config any) model.ExperimentMeasurer {
				return telegram.NewExperimentMeasurer(
					config.(telegram.Config),
				)
			},
			buildRicherInputExperiment: telegram.NewRicherInputExperiment,
			config:                     telegram.Config{},
			enabledByDefault:           true,
			interruptible:              false,
			inputPolicy:                model.InputNone,
		}
	}
}
