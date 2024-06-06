package registry

//
// Registers the `simple sni' experiment from the dslx tutorial.
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tutorial/dslx/chapter02"
)

func init() {
	const canonicalName = "simple_sni"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return chapter02.NewExperimentMeasurer(
					*config.(*chapter02.Config),
				)
			},
			canonicalName: canonicalName,
			config:        &chapter02.Config{},
			inputPolicy:   model.InputOrQueryBackend,
		}
	}
}
