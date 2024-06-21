package nettests

import "github.com/ooni/probe-cli/v3/internal/model"

// Signal nettest implementation.
type Signal struct{}

// Run starts the nettest.
func (h Signal) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"signal",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []model.ExperimentTarget{model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("")})
}
