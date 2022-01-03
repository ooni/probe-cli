package nettests

import (
	"context"

	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// STUNReachability nettest implementation.
type STUNReachability struct{}

func (n STUNReachability) lookupURLs(ctl *Controller) ([]string, error) {
	inputloader := &engine.InputLoader{
		CheckInConfig: &model.OOAPICheckInConfig{
			// not needed because we have default static input in the engine
		},
		ExperimentName: "stunreachability",
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
func (n STUNReachability) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("stunreachability")
	if err != nil {
		return err
	}
	urls, err := n.lookupURLs(ctl)
	if err != nil {
		return err
	}
	return ctl.Run(builder, urls)
}
