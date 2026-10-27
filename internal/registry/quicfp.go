package registry

//
// Registers the `quicfp' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/quicfp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "quicfp"
	register(canonicalName, func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return quicfp.NewExperimentMeasurer()
			},
			canonicalName: canonicalName,
			config:        &quicfp.Config{},
			// not enabled by default: it is new and has not been reviewed yet
			enabledByDefault: false,
			inputPolicy:      model.InputStrictlyRequired,
			newLoader:        quicfp.NewLoader,
		}
	})
}
