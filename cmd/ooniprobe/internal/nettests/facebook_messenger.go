package nettests

import "github.com/ooni/probe-cli/v3/internal/model"

// FacebookMessenger test implementation
type FacebookMessenger struct {
}

// Run starts the test
func (h FacebookMessenger) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"facebook_messenger",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []model.ExperimentTarget{model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("")})
}
