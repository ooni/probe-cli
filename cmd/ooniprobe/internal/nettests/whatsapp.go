package nettests

import "github.com/ooni/probe-cli/v3/internal/model"

// WhatsApp test implementation
type WhatsApp struct {
}

// Run starts the test
func (h WhatsApp) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"whatsapp",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []model.ExperimentTarget{model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("")})
}
