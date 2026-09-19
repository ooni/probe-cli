package registry

//
// Registers the `psiphon' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/psiphon"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "psiphon"
	register(canonicalName, func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return psiphon.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &psiphon.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOptional,
			newLoader:        psiphon.NewLoader,
		}
	})
}
