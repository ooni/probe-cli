package registry

//
// Registers the `psiphon' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/psiphon"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["psiphon"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return psiphon.NewExperimentMeasurer(
				*config.(*psiphon.Config),
			)
		},
		config:      &psiphon.Config{},
		inputPolicy: model.InputOptional,
	}
}
