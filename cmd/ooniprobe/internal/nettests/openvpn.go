package nettests

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// OpenVPN nettest implementation.
type OpenVPN struct{}

func (o OpenVPN) loadTargets(ctl *Controller, builder model.ExperimentBuilder) ([]model.ExperimentTarget, error) {
	config := &model.ExperimentTargetLoaderConfig{
		CheckInConfig: &model.OOAPICheckInConfig{},
		Session:       ctl.Session,
		SourceFiles:   ctl.InputFiles,
		StaticInputs:  ctl.Inputs,
	}
	targetloader := builder.NewTargetLoader(config)
	targets, err := targetloader.Load(context.Background())
	if err != nil {
		return nil, err
	}
	return ctl.BuildAndSetInputIdxMap(targets)
}

// Run starts the nettest.
func (o OpenVPN) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("openvpn")
	if err != nil {
		return err
	}
	inputs, err := o.loadTargets(ctl, builder)
	if err != nil {
		return err
	}
	return ctl.Run(builder, inputs)
}
