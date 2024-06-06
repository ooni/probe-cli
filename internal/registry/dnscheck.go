package registry

//
// Registers the `dnscheck' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "dnscheck"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return dnscheck.NewExperimentMeasurer(
					*config.(*dnscheck.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &dnscheck.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
		}
	}
}
