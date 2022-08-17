package registry

//
// Registers the `web_connectivity' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["web_connectivity"] = &Factory{
		build: func(config any) model.ExperimentMeasurer {
			return webconnectivity.NewExperimentMeasurer(
				config.(webconnectivity.Config),
			)
		},
		config:        &webconnectivity.Config{},
		interruptible: false,
		inputPolicy:   model.InputOrQueryBackend,
	}
}
