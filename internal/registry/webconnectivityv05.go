package registry

//
// Registers the `web_connectivity@v0.5' experiment.
//
// See https://github.com/ooni/probe/issues/2237
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["web_connectivity@v0.5"] = &Factory{
		build: func(config any) model.ExperimentMeasurer {
			return webconnectivity.NewExperimentMeasurer(
				config.(*webconnectivity.Config),
			)
		},
		config:        &webconnectivity.Config{},
		interruptible: false,
		inputPolicy:   model.InputOrQueryBackend,
	}
}
