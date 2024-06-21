package nettests

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func (n WebConnectivity) lookupURLs(
	ctl *Controller, builder model.ExperimentBuilder, categories []string) ([]model.ExperimentTarget, error) {
	config := &model.ExperimentTargetLoaderConfig{
		CheckInConfig: &model.OOAPICheckInConfig{
			// Setting Charging and OnWiFi to true causes the CheckIn
			// API to return to us as much URL as possible with the
			// given RunType hint.
			Charging: true,
			OnWiFi:   true,
			RunType:  ctl.RunType,
			WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
				CategoryCodes: categories,
			},
		},
		Session:      ctl.Session,
		SourceFiles:  ctl.InputFiles,
		StaticInputs: ctl.Inputs,
	}
	targetloader := builder.NewTargetLoader(config)
	testlist, err := targetloader.Load(context.Background())
	if err != nil {
		return nil, err
	}
	return ctl.BuildAndSetInputIdxMap(testlist)
}

// WebConnectivity test implementation
type WebConnectivity struct{}

// Run starts the test
func (n WebConnectivity) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("web_connectivity")
	if err != nil {
		return err
	}
	log.Debugf("Enabled category codes are the following %v", ctl.Probe.Config().Nettests.WebsitesEnabledCategoryCodes)
	urls, err := n.lookupURLs(ctl, builder, ctl.Probe.Config().Nettests.WebsitesEnabledCategoryCodes)
	if err != nil {
		return err
	}
	return ctl.Run(builder, urls)
}
