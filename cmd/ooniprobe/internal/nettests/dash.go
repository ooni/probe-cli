package nettests

import "github.com/ooni/probe-cli/v3/internal/model"

// Dash test implementation
type Dash struct {
}

// Run starts the test
func (d Dash) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("dash")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []model.ExperimentTarget{model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("")})
}
