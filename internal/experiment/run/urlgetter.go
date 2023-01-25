package run

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type urlGetterMain struct{}

func (m *urlGetterMain) do(ctx context.Context, input StructuredInput,
	sess model.ExperimentSession, measurement *model.Measurement,
	callbacks model.ExperimentCallbacks) error {
	exp := urlgetter.Measurer{
		Config: input.URLGetter,
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
