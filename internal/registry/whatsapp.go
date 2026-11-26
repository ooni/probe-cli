package registry

//
// Registers the `whatsapp' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/whatsapp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "whatsapp"
	register(canonicalName, "v0.11", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return whatsapp.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &whatsapp.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
			newLoader:        whatsapp.NewLoader,
		}
	})
}
