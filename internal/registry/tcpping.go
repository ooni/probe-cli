package registry

//
// Registers the `tcpping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tcpping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "tcpping"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return tcpping.NewExperimentMeasurer(
					*config.(*tcpping.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &tcpping.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		}
	}
}
