package registry

//
// Registers the `ndt' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/ndt7"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["ndt"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return ndt7.NewExperimentMeasurer(
				*config.(*ndt7.Config),
			)
		},
		config:        &ndt7.Config{},
		interruptible: true,
		inputPolicy:   model.InputNone,
	}
}
