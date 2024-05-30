package registry

//
// Registers the `simplequicping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/simplequicping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["simplequicping"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
				return simplequicping.NewExperimentMeasurer(
					*config.(*simplequicping.Config),
				)
			},
			buildRicherInputExperiment: simplequicping.NewRicherInputExperiment,
			config:                     &simplequicping.Config{},
			enabledByDefault:           true,
			inputPolicy:                model.InputStrictlyRequired,
		}
	}
}
