package run

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
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
	return exp.Run(ctx, sess, measurement, callbacks)
}
