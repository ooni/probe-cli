package mocks

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Experiment mocks model.Experiment
type Experiment struct {
	MockKibiBytesReceived func() float64

	MockKibiBytesSent func() float64

	MockName func() string

	MockGetSummaryKeys func(m *model.Measurement) (any, error)

	MockReportID func() string

	MockMeasureAsync func(ctx context.Context, input string) (<-chan *model.Measurement, error)

	MockMeasureWithContext func(
		ctx context.Context, input string) (measurement *model.Measurement, err error)

	MockSaveMeasurement func(measurement *model.Measurement, filePath string) error

	MockSubmitAndUpdateMeasurementContext func(
		ctx context.Context, measurement *model.Measurement) error

	MockOpenReportContext func(ctx context.Context) error
}

func (e *Experiment) KibiBytesReceived() float64 {
	return e.MockKibiBytesReceived()
}

func (e *Experiment) KibiBytesSent() float64 {
	return e.MockKibiBytesSent()
}

func (e *Experiment) Name() string {
	return e.MockName()
}

func (e *Experiment) GetSummaryKeys(m *model.Measurement) (any, error) {
	return e.MockGetSummaryKeys(m)
}

func (e *Experiment) ReportID() string {
	return e.MockReportID()
}

func (e *Experiment) MeasureAsync(
	ctx context.Context, input string) (<-chan *model.Measurement, error) {
	return e.MockMeasureAsync(ctx, input)
}

func (e *Experiment) MeasureWithContext(
	ctx context.Context, input string) (measurement *model.Measurement, err error) {
	return e.MockMeasureWithContext(ctx, input)
}

func (e *Experiment) SaveMeasurement(measurement *model.Measurement, filePath string) error {
	return e.MockSaveMeasurement(measurement, filePath)
}

func (e *Experiment) SubmitAndUpdateMeasurementContext(
	ctx context.Context, measurement *model.Measurement) error {
	return e.MockSubmitAndUpdateMeasurementContext(ctx, measurement)
}

func (e *Experiment) OpenReportContext(ctx context.Context) error {
	return e.MockOpenReportContext(ctx)
}
