package registry

//
// Registers the `wireguard` experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/wireguard"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["wireguard"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return wireguard.NewExperimentMeasurer(
				*config.(*wireguard.Config),
			)
		},
		config:      &wireguard.Config{},
		inputPolicy: model.InputStrictlyRequired,
	}
}
