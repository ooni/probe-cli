package mocks

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// RicherInputExperiment mocks model.RicherInputExperiment
type RicherInputExperiment struct {
	MockKibiBytesReceived func() float64

	MockKibiBytesSent func() float64

	MockName func() string

	MockMeasure func(ctx context.Context, input model.RicherInput) (*model.Measurement, error)

	MockNewReportTemplate func() *model.OOAPIReportTemplate
}

func (e *RicherInputExperiment) KibiBytesReceived() float64 {
	return e.MockKibiBytesReceived()
}

func (e *RicherInputExperiment) KibiBytesSent() float64 {
	return e.MockKibiBytesSent()
}

func (e *RicherInputExperiment) Name() string {
	return e.MockName()
}

func (e *RicherInputExperiment) Measure(ctx context.Context, input model.RicherInput) (*model.Measurement, error) {
	return e.MockMeasure(ctx, input)
}

func (e *RicherInputExperiment) NewReportTemplate() *model.OOAPIReportTemplate {
	return e.MockNewReportTemplate()
}
