package registry

//
// Registers the `tlsmiddlebox' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tlsmiddlebox"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["tlsmiddlebox"] = &Factory{
		buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
			return tlsmiddlebox.NewExperimentMeasurer(
				*config.(*tlsmiddlebox.Config),
			)
		},
		buildRicherInputExperiment: tlsmiddlebox.NewRicherInputExperiment,
		config:                     &tlsmiddlebox.Config{},
		enabledByDefault:           true,
		inputPolicy:                model.InputStrictlyRequired,
	}
}
