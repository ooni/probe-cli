package registry

//
// Registers the `telegram' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "telegram"
	register(canonicalName, func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return telegram.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &telegram.Config{},
			enabledByDefault: true,
			interruptible:    false,
			inputPolicy:      model.InputNone,
			newLoader:        telegram.NewLoader,
		}
	})
}
