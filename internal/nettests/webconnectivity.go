package nettests

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/session"
)

// WebConnectivityFactoryConfig contains config for creating
// a new [WebConnectivityFactory] instance.
type WebConnectivityFactoryConfig struct {
	// CheckIn contains the MANDATORY response returned by check-in.
	CheckIn *model.OOAPICheckInResult

	// InputFiles contains the OPTIONAL input files to read.
	InputFiles []string

	// Inputs contains the OPTIONAL inputs to read.
	Inputs []string

	// Session is the MANDATORY session.
	Session *Session
}

// WebConnectivityFactory creates the [WebConnectivity] experiment. The zero
// value is invalid. Construct using [NewWebConnectivityFactory].
type WebConnectivityFactory struct {
	experiment *WebConnectivity
}

// NewWebConnectivityFactory creates a new [WebConnectivity] factory.
func NewWebConnectivityFactory(config *WebConnectivityFactoryConfig) (*WebConnectivityFactory, error) {
	runtimex.Assert(config != nil, "passed nil config")
	runtimex.Assert(config.CheckIn != nil, "passed nil config.CheckIn")
	runtimex.Assert(config.Session != nil, "passed nil config.Session")
	if config.CheckIn.Tests.WebConnectivity == nil {
		return nil, ErrMissingCheckInConfig
	}
	f := &WebConnectivityFactory{
		experiment: &WebConnectivity{
			byteCounter:   bytecounter.New(),
			config:        config,
			callbacks:     model.NewPrinterCallbacks(model.DiscardLogger),
			testStartTime: time.Now(),
		},
	}
	return f, nil
}

var _ ExperimentFactory = &WebConnectivityFactory{}

// LoadInputs loads the proper inputs for this experiment.
func (f *WebConnectivityFactory) LoadInputs() ([]model.OOAPIURLInfo, error) {
	return loadInputs(
		f.experiment.config.CheckIn.Tests.WebConnectivity.URLs,
		f.experiment.config.InputFiles,
		f.experiment.config.Inputs,
	)
}

// NewExperiment implements ExperimentFactory
func (f *WebConnectivityFactory) NewExperiment(callbacks model.ExperimentCallbacks) Experiment {
	f.experiment.callbacks = callbacks
	return f.experiment
}

// WebConnectivity is the Web Connectivity experiment. The zero value
// is invalid. Construct using [WebConnectivityFactory].
type WebConnectivity struct {
	// byteCounter is the byte counter we use.
	byteCounter *bytecounter.Counter

	// config is the config with which we were created.
	config *WebConnectivityFactoryConfig

	// callbacks contains the experiment callbacks.
	callbacks model.ExperimentCallbacks

	// testStartTime is when we started this test.
	testStartTime time.Time
}

var _ Experiment = &WebConnectivity{}

// GetSummaryKeys implements Experiment
func (e *WebConnectivity) GetSummaryKeys(m *model.Measurement) (any, error) {
	return webconnectivity.GetSummaryKeys(m)
}

// KibiBytesReceived implements Experiment
func (e *WebConnectivity) KibiBytesReceived() float64 {
	return e.byteCounter.KibiBytesReceived()
}

// KibiBytesSent implements Experiment
func (e *WebConnectivity) KibiBytesSent() float64 {
	return e.byteCounter.KibiBytesSent()
}

// Measure implements Experiment
func (e *WebConnectivity) Measure(ctx context.Context, input string) (*model.Measurement, error) {
	// TODO(bassosimone): how to measure the bytes sent and received?
	req := &session.WebConnectivityRequest{
		Input:         input,
		ReportID:      e.config.CheckIn.Tests.WebConnectivity.ReportID,
		TestStartTime: e.testStartTime,
	}
	runtimex.Assert(e.config.Session.session != nil, "expected non-nil Session.session")
	sess := e.config.Session
	if err := sess.session.Send(ctx, &session.Request{WebConnectivity: req}); err != nil {
		return nil, err
	}
	for {
		resp, err := sess.session.Recv(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Log != nil {
			sess.emitLog(resp.Log)
			continue
		}
		if resp.WebConnectivity != nil {
			return resp.WebConnectivity.Measurement, resp.WebConnectivity.Error
		}
	}
}

// Name implements Experiment
func (e *WebConnectivity) Name() string {
	return webconnectivity.ExperimentName
}

// ReportID implements Experiment
func (e *WebConnectivity) ReportID() string {
	return e.config.CheckIn.Tests.WebConnectivity.ReportID
}
