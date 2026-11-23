package registry

//
// Registers the `signal' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "signal"
	register(canonicalName, "v0.2", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return signal.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &signal.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
			newLoader:        signal.NewLoader,
		}
	})
}
