package nettests

import "github.com/ooni/probe-cli/v3/internal/model"

// TorSf test implementation
type TorSf struct {
}

// Run starts the test
func (h TorSf) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("torsf")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []model.ExperimentTarget{model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("")})
}

func (h TorSf) onlyBackground() {}
