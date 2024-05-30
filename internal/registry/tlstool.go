package registry

//
// Registers the `tlstool' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tlstool"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["tlstool"] = &Factory{
		buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
			return tlstool.NewExperimentMeasurer(
				*config.(*tlstool.Config),
			)
		},
		buildRicherInputExperiment: tlstool.NewRicherInputExperiment,
		config:                     &tlstool.Config{},
		enabledByDefault:           true,
		inputPolicy:                model.InputOrQueryBackend,
	}
}
