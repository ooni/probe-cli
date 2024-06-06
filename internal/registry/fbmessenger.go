package registry

//
// Registers the `fbmessenger' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "facebook_messenger"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return fbmessenger.NewExperimentMeasurer(
					*config.(*fbmessenger.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &fbmessenger.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		}
	}
}
