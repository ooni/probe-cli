package registry

//
// Registers the `tlsping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tlsping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "tlsping"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return tlsping.NewExperimentMeasurer(
					*config.(*tlsping.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &tlsping.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		}
	}
}
