package nettests

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/database"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// TODO(bassosimone): we should propagate the kind of run
// to here such that we get the right runType.

func lookupURLs(ctl *Controller, categories []string) ([]string, map[int64]int64, error) {
	inputloader := &engine.InputLoader{
		CheckInConfig: &model.CheckInConfig{
			WebConnectivity: model.CheckInConfigWebConnectivity{
				CategoryCodes: categories,
			},
		},
		InputPolicy:  engine.InputOrQueryBackend,
		Session:      ctl.Session,
		SourceFiles:  ctl.InputFiles,
		StaticInputs: ctl.Inputs,
	}
	testlist, err := inputloader.Load(context.Background())
	var urls []string
	urlIDMap := make(map[int64]int64)
	if err != nil {
		return nil, nil, err
	}
	for idx, url := range testlist {
		log.Debugf("Going over URL %d", idx)
		urlID, err := database.CreateOrUpdateURL(
			ctl.Probe.DB(), url.URL, url.CategoryCode, url.CountryCode,
		)
		if err != nil {
			log.Error("failed to add to the URL table")
			return nil, nil, err
		}
		log.Debugf("Mapped URL %s to idx %d and urlID %d", url.URL, idx, urlID)
		urlIDMap[int64(idx)] = urlID
		urls = append(urls, url.URL)
	}
	return urls, urlIDMap, nil
}

// WebConnectivity test implementation
type WebConnectivity struct{}

// Run starts the test
func (n WebConnectivity) Run(ctl *Controller) error {
	log.Debugf("Enabled category codes are the following %v", ctl.Probe.Config().Nettests.WebsitesEnabledCategoryCodes)
	urls, urlIDMap, err := lookupURLs(ctl, ctl.Probe.Config().Nettests.WebsitesEnabledCategoryCodes)
	if err != nil {
		return err
	}
	ctl.SetInputIdxMap(urlIDMap)
	builder, err := ctl.Session.NewExperimentBuilder(
		"web_connectivity",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, urls)
}
