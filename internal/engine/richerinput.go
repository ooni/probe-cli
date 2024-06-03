package engine

//
// Richer input
//
// See XXX
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/erroror"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/registry"
)

// NewRicherInputExperiment constructs a new richer-input experiment.
func (sess *Session) NewRicherInputExperiment(name string) (model.RicherInputExperiment, error) {
	factory, err := registry.NewFactory(name, sess.kvStore, sess.logger)
	if err != nil {
		return nil, err
	}
	return factory.NewRicherInputExperiment(sess)
}

// richerInputExperiment wraps a [model.RicherInputExperiment] making sure
// we account bytes I/O using the session's byte counter.
type richerInputExperiment struct {
	exp  model.RicherInputExperiment
	sess *Session
}

var _ model.RicherInputExperiment = &richerInputExperiment{}

// KibiBytesReceived implements model.RicherInputExperiment.
func (r *richerInputExperiment) KibiBytesReceived() float64 {
	return r.exp.KibiBytesReceived()
}

// KibiBytesSent implements model.RicherInputExperiment.
func (r *richerInputExperiment) KibiBytesSent() float64 {
	return r.exp.KibiBytesSent()
}

// Name implements model.RicherInputExperiment.
func (r *richerInputExperiment) Name() string {
	return r.exp.Name()
}

// OpenReport implements model.RicherInputExperiment.
func (r *richerInputExperiment) OpenReport(ctx context.Context) error {
	ctx = bytecounter.WithSessionByteCounter(ctx, r.sess.byteCounter)
	return r.exp.OpenReport(ctx)
}

// ReportID implements model.RicherInputExperiment.
func (r *richerInputExperiment) ReportID() string {
	return r.exp.ReportID()
}

// Start implements model.RicherInputExperiment.
func (r *richerInputExperiment) Start(ctx context.Context, config *model.RicherInputConfig) <-chan *erroror.Value[*model.Measurement] {
	ctx = bytecounter.WithSessionByteCounter(ctx, r.sess.byteCounter)
	return r.exp.Start(ctx, config)
}

// SubmitMeasurement implements model.RicherInputExperiment.
func (r *richerInputExperiment) SubmitMeasurement(ctx context.Context, m *model.Measurement) error {
	// no need to wrap the context here since r.exp internally uses a model.OOAPIReport
	// that has been created with a context using the session's byte counter
	return r.exp.SubmitMeasurement(ctx, m)
}
