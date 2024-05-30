package registry

//
// Registers the `echcheck' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/echcheck"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["echcheck"] = &Factory{
		buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
			return echcheck.NewExperimentMeasurer(
				*config.(*echcheck.Config),
			)
		},
		buildRicherInputExperiment: echcheck.NewRicherInputExperiment,
		config:                     &echcheck.Config{},
		inputPolicy:                model.InputOptional,
	}
}
