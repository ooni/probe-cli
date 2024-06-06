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
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return whatsapp.NewExperimentMeasurer(
					*config.(*whatsapp.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &whatsapp.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		}
	}
}
