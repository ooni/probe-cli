package registry

//
// Registers the `echcheck' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/echcheck"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "echcheck"
	register(canonicalName, "v0.3", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return echcheck.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &echcheck.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOptional,
			newLoader:        echcheck.NewLoader,
		}
	})
}
