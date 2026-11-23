package registry

//
// Registers the `dnsping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/dnsping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "dnsping"
	register(canonicalName, "v0.4", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return dnsping.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &dnsping.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
			newLoader:        dnsping.NewLoader,
		}
	})
}
