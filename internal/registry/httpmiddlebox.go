package registry

//
// Registers the `httpmiddlebox' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/httpmiddlebox"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "httpmiddlebox"
	register(canonicalName, func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return httpmiddlebox.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &httpmiddlebox.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
			newLoader:        httpmiddlebox.NewLoader,
		}
	})
}
