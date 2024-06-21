package registry

//
// Registers the `httphostheader' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/httphostheader"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "http_host_header"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return httphostheader.NewExperimentMeasurer(
					*config.(*httphostheader.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &httphostheader.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
		}
	}
}
