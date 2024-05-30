package registry

//
// Registers the `tcpping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tcpping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["tcpping"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
				return tcpping.NewExperimentMeasurer(
					*config.(*tcpping.Config),
				)
			},
			buildRicherInputExperiment: tcpping.NewRicherInputExperiment,
			config:                     &tcpping.Config{},
			enabledByDefault:           true,
			inputPolicy:                model.InputStrictlyRequired,
		}
	}
}
