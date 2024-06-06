package registry

//
// Registers the `openvpn' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/openvpn"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["openvpn"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return openvpn.NewExperimentMeasurer(
					*config.(*openvpn.Config), "openvpn",
				)
			},
			config:           &openvpn.Config{},
			enabledByDefault: true,
			interruptible:    true,
			inputPolicy:      model.InputOrQueryBackend,
		}
	}
}
