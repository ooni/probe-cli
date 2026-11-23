package registry

//
// Registers the `urlgetter' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "urlgetter"
	register(canonicalName, "v0.2", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return urlgetter.NewExperimentMeasurer(
					*config.(*urlgetter.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &urlgetter.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		}
	})
}
