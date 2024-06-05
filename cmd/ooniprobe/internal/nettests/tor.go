package nettests

import "github.com/ooni/probe-cli/v3/internal/model"

// Tor test implementation
type Tor struct {
}

// Run starts the test
func (h Tor) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"tor",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []model.ExperimentTarget{model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("")})
}
