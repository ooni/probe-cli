package registry

//
// Registers the `web_connectivity@v0.5' experiment.
//
// See https://github.com/ooni/probe/issues/2237
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	// Register web_connectivity LTE as the "v0.5" version of the
	// "web_connectivity" experiment. The registration stamps the base name as
	// the canonical name, so "@v0.5" never leaks into the measurement.
	registerVersion("web_connectivity", "v0.5", func() *Factory {
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
	})
}
