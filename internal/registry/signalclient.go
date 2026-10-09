package registry

//
// Registers the `signal_client' experiment (provisional name).
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/signalclient"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "signal_client"
	register(canonicalName, func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return signalclient.NewExperimentMeasurer(*config.(*signalclient.Config))
			},
			canonicalName:    canonicalName,
			config:           &signalclient.Config{},
			enabledByDefault: false,
			inputPolicy:      model.InputNone,
		}
	})
}
