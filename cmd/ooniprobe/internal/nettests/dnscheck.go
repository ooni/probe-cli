package nettests

import (
	"context"

	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// DNSCheck nettest implementation.
type DNSCheck struct{}

func (n DNSCheck) lookupURLs(ctl *Controller) ([]string, error) {
	inputloader := &engine.InputLoader{
		CheckInConfig: &model.CheckInConfig{
			// not needed because we have default static input in the engine
		},
		ExperimentName: "dnscheck",
		InputPolicy:    engine.InputOrStaticDefault,
		Session:        ctl.Session,
		SourceFiles:    ctl.InputFiles,
		StaticInputs:   ctl.Inputs,
	}
	testlist, err := inputloader.Load(context.Background())
	if err != nil {
		return nil, err
	}
	return ctl.BuildAndSetInputIdxMap(ctl.Probe.DB(), testlist)
}

// Run starts the nettest.
func (n DNSCheck) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("dnscheck")
	if err != nil {
		return err
	}
	urls, err := n.lookupURLs(ctl)
	if err != nil {
		return err
	}
	return ctl.Run(builder, urls)
}
