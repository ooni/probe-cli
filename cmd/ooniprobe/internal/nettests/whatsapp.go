package nettests

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// WhatsApp test implementation
type WhatsApp struct {
}

func (h WhatsApp) lookupURLs(ctl *Controller, builder model.ExperimentBuilder) ([]model.ExperimentTarget, error) {
	config := &model.ExperimentTargetLoaderConfig{
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

// Run starts the test
func (h WhatsApp) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"whatsapp",
	)
	if err != nil {
		return err
	}
	urls, err := h.lookupURLs(ctl, builder)
	if err != nil {
		return err
	}
	return ctl.Run(builder, urls)
}
