package nettests

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/config"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func newCheckInConfig(runType model.RunType, config *config.Config) *model.OOAPICheckInConfig {
	result := &model.OOAPICheckInConfig{
		// Setting Charging and OnWiFi to true causes the CheckIn
		// API to return to us as much URL as possible with the
		// given RunType hint.
		Charging: true,
		OnWiFi:   true,
		RunType:  runType,
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: config.Nettests.WebsitesEnabledCategoryCodes,
		},
	}
	if result.WebConnectivity.CategoryCodes == nil {
		result.WebConnectivity.CategoryCodes = []string{}
	}
	return result
}

func (n WebConnectivity) lookupURLs(ctl *Controller, config *config.Config) ([]string, error) {
	inputloader := &engine.InputLoader{
		CheckInConfig:  newCheckInConfig(ctl.RunType, config),
		ExperimentName: "web_connectivity",
		InputPolicy:    model.InputOrQueryBackend,
		Session:        ctl.Session,
		SourceFiles:    ctl.InputFiles,
		StaticInputs:   ctl.Inputs,
	}
	testlist, err := inputloader.Load(context.Background())
	if err != nil {
		return nil, err
	}
	return ctl.BuildAndSetInputIdxMap(testlist)
}

// WebConnectivity test implementation
type WebConnectivity struct{}

// Run starts the test
func (n WebConnectivity) Run(ctl *Controller) error {
	log.Debugf("Enabled category codes are the following %v", ctl.Probe.Config().Nettests.WebsitesEnabledCategoryCodes)
	urls, err := n.lookupURLs(ctl, ctl.Probe.Config())
	if err != nil {
		return err
	}
	builder, err := ctl.Session.NewExperimentBuilder("web_connectivity")
	if err != nil {
		return err
	}
	return ctl.Run(builder, urls)
}
