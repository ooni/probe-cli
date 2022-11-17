package run

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type dnsCheckMain struct {
	Endpoints *dnscheck.Endpoints
}

func (m *dnsCheckMain) do(ctx context.Context, input StructuredInput,
	sess model.ExperimentSession, measurement *model.Measurement,
	callbacks model.ExperimentCallbacks) error {
	exp := dnscheck.Measurer{
		Config:    input.DNSCheck,
		Endpoints: m.Endpoints,
	}
	measurement.TestName = exp.ExperimentName()
	measurement.TestVersion = exp.ExperimentVersion()
	measurement.Input = model.MeasurementTarget(input.Input)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	return exp.Run(ctx, args)
}
