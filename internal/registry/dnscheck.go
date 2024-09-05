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
		// TODO(bassosimone,DecFox): for now, we MUST keep the InputOrStaticDefault
		// policy because otherwise ./pkg/oonimkall should break.
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return dnscheck.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &dnscheck.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
			newLoader:        dnscheck.NewLoader,
		}
	}
}
