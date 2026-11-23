package registry

//
// Registers the `simplequicping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/simplequicping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "simplequicping"
	register(canonicalName, "v0.2", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return simplequicping.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &simplequicping.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
			newLoader:        simplequicping.NewLoader,
		}
	})
}
