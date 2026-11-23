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
	register(canonicalName, "v0.2", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return fbmessenger.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &fbmessenger.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
			newLoader:        fbmessenger.NewLoader,
		}
	})
}
