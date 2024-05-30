package registry

//
// Registers the `dnscheck' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["dnscheck"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
				return dnscheck.NewExperimentMeasurer(
					*config.(*dnscheck.Config),
				)
			},
			buildRicherInputExperiment: dnscheck.NewRicherInputExperiment,
			config:                     &dnscheck.Config{},
			enabledByDefault:           true,
			inputPolicy:                model.InputOrStaticDefault,
		}
	}
}
