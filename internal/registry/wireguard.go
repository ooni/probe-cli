package registry

//
// Registers the `wireguard' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/wireguard"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "wireguard"
	AllExperiments["wireguard"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return wireguard.NewExperimentMeasurer(
					*config.(*wireguard.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &wireguard.Config{},
			enabledByDefault: true,
			interruptible:    true,
			inputPolicy:      model.InputNone,
		}
	}
}
