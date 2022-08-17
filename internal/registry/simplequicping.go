package registry

//
// Registers the `simplequicping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/simplequicping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["simplequicping"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return simplequicping.NewExperimentMeasurer(
				*config.(*simplequicping.Config),
			)
		},
		config:      &simplequicping.Config{},
		inputPolicy: model.InputStrictlyRequired,
	}
}
