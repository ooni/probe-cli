package registry

//
// Registers the `tor' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tor"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["tor"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
				return tor.NewExperimentMeasurer(
					*config.(*tor.Config),
				)
			},
			buildRicherInputExperiment: tor.NewRicherInputExperiment,
			config:                     &tor.Config{},
			enabledByDefault:           true,
			inputPolicy:                model.InputNone,
		}
	}
}
