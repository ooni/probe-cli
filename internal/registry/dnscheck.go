package registry

//
// Registers the `dnscheck' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["dnscheck"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return dnscheck.NewExperimentMeasurer(
				*config.(*dnscheck.Config),
			)
		},
		config:      &dnscheck.Config{},
		inputPolicy: model.InputOrStaticDefault,
	}
}
