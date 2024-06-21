package registry

//
// Registers the 'portfiltering' experiment
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/portfiltering"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "portfiltering"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return portfiltering.NewExperimentMeasurer(
					config.(portfiltering.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           portfiltering.Config{},
			enabledByDefault: true,
			interruptible:    false,
			inputPolicy:      model.InputNone,
		}
	}
}
