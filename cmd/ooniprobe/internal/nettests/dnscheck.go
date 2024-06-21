package nettests

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSCheck nettest implementation.
type DNSCheck struct{}

func (n DNSCheck) lookupURLs(ctl *Controller, builder model.ExperimentBuilder) ([]model.ExperimentTarget, error) {
	config := &model.ExperimentTargetLoaderConfig{
		CheckInConfig: &model.OOAPICheckInConfig{
			// not needed because we have default static input in the engine
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

// Run starts the nettest.
func (n DNSCheck) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("dnscheck")
	if err != nil {
		return err
	}
	urls, err := n.lookupURLs(ctl, builder)
	if err != nil {
		return err
	}
	return ctl.Run(builder, urls)
}
