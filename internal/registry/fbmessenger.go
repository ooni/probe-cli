package registry

//
// Registers the `fbmessenger' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["facebook_messenger"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return fbmessenger.NewExperimentMeasurer(
					*config.(*fbmessenger.Config),
				)
			},
			config:           &fbmessenger.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		}
	}
}
