package registry

//
// Registers the `sni_throttling' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/snithrottling"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "sni_throttling"
	register(canonicalName, func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return snithrottling.NewExperimentMeasurer(
					*config.(*snithrottling.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &snithrottling.Config{},
			enabledByDefault: true,
			interruptible:    false,
			inputPolicy:      model.InputNone,
		}
	})
}
