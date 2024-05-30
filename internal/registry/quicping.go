package registry

//
// Registers the `quicping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/quicping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["quicping"] = &Factory{
		buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
			return quicping.NewExperimentMeasurer(
				*config.(*quicping.Config),
			)
		},
		buildRicherInputExperiment: quicping.NewRicherInputExperiment,
		config:                     &quicping.Config{},
		enabledByDefault:           true,
		inputPolicy:                model.InputStrictlyRequired,
	}
}
