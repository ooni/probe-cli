package registry

//
// Registers the `web_connectivity' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "web_connectivity"
	registerExperiment(canonicalName, "v0.4", map[string]func() *Factory{
		"v0.4": func() *Factory {
			return &Factory{
				build: func(config any) model.ExperimentMeasurer {
					return webconnectivity.NewExperimentMeasurer()
				},
				canonicalName:    canonicalName,
				config:           &webconnectivity.Config{},
				enabledByDefault: true,
				interruptible:    false,
				inputPolicy:      model.InputOrQueryBackend,
				newLoader:        webconnectivity.NewLoader,
			}
		},
		"v0.5": func() *Factory {
			return &Factory{
				build: func(config any) model.ExperimentMeasurer {
					return webconnectivitylte.NewExperimentMeasurer()
				},
				config:           &webconnectivitylte.Config{},
				enabledByDefault: true,
				interruptible:    false,
				inputPolicy:      model.InputOrQueryBackend,
				newLoader:        webconnectivitylte.NewLoader,
			}
		},
	})
}
