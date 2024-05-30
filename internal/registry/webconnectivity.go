package registry

//
// Registers the `web_connectivity' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["web_connectivity"] = &Factory{
		buildMeasurer: func(config any) model.ExperimentMeasurer {
			return webconnectivity.NewExperimentMeasurer(
				config.(webconnectivity.Config),
			)
		},
		buildRicherInputExperiment: webconnectivity.NewRicherInputExperiment,
		config:                     webconnectivity.Config{},
		enabledByDefault:           true,
		interruptible:              false,
		inputPolicy:                model.InputOrQueryBackend,
	}
}
