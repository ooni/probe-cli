package registry

//
// Registers the `echcheck' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/echcheck"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["echcheck"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return echcheck.NewExperimentMeasurer(
				*config.(*echcheck.Config),
			)
		},
		config:      &echcheck.Config{},
		inputPolicy: model.InputOptional,
	}
}
