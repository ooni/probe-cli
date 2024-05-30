package engine

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// richerInputExperimentWrapper wraps a [model.RicherInputExperiment] to
// account for the bytes sent and received into the session's counter.
type richerInputExperimentWrapper struct {
	exp  model.RicherInputExperiment
	sess *Session
}

var _ model.RicherInputExperiment = &richerInputExperimentWrapper{}

// KibiBytesReceived implements model.RicherInputExperiment.
func (r *richerInputExperimentWrapper) KibiBytesReceived() float64 {
	return r.exp.KibiBytesReceived()
}

// KibiBytesSent implements model.RicherInputExperiment.
func (r *richerInputExperimentWrapper) KibiBytesSent() float64 {
	return r.exp.KibiBytesSent()
}

// Measure implements model.RicherInputExperiment.
func (r *richerInputExperimentWrapper) Measure(ctx context.Context, input model.RicherInput) (*model.Measurement, error) {
	// make sure we account bytes into the session's byte counter
	ctx = bytecounter.WithSessionByteCounter(ctx, r.sess.byteCounter)
	return r.exp.Measure(ctx, input)
}

// Name implements model.RicherInputExperiment.
func (r *richerInputExperimentWrapper) Name() string {
	return r.exp.Name()
}

// NewReportTemplate implements model.RicherInputExperiment.
func (r *richerInputExperimentWrapper) NewReportTemplate() *model.OOAPIReportTemplate {
	return r.exp.NewReportTemplate()
}
