package nettests

import "github.com/ooni/probe-cli/v3/internal/model"

// ECHCheck nettest implementation.
type ECHCheck struct{}

// Run starts the nettest.
func (n ECHCheck) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("echcheck")
	if err != nil {
		return err
	}
	// providing an input containing an empty string causes the experiment
	// to recognize the empty string and use the default URL
	return ctl.Run(builder, []model.ExperimentTarget{
		model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("https://cloudflare-ech.com/cdn-cgi/trace"),
		model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("https://cloudflare-ech.com:443"),
		model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("https://min-ng.test.defo.ie:15443"),
	})
}
