package registry

//
// Registers the `quicmiddlebox' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/quicmiddlebox"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "quicmiddlebox"
	register(canonicalName, func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return quicmiddlebox.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &quicmiddlebox.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
			newLoader:        quicmiddlebox.NewLoader,
		}
	})
}
