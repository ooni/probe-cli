package registry

//
// Registers the `stunreachability' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/stunreachability"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "stunreachability"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return stunreachability.NewExperimentMeasurer(
					*config.(*stunreachability.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &stunreachability.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
		}
	}
}
