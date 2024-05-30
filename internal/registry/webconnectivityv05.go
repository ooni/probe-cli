package registry

//
// Registers the `web_connectivity@v0.5' experiment.
//
// See https://github.com/ooni/probe/issues/2237
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["web_connectivity@v0.5"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config any) model.ExperimentMeasurer {
				return webconnectivitylte.NewExperimentMeasurer(
					config.(*webconnectivitylte.Config),
				)
			},
			buildRicherInputExperiment: webconnectivitylte.NewRicherInputExperiment,
			config:                     &webconnectivitylte.Config{},
			enabledByDefault:           true,
			interruptible:              false,
			inputPolicy:                model.InputOrQueryBackend,
		}
	}
}
