package run

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type dnsCheckMain struct {
	Endpoints *dnscheck.Endpoints
	mu        sync.Mutex
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
	return exp.Run(ctx, sess, measurement, callbacks)
}
