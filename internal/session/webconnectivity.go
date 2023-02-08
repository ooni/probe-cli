package session

//
// Running the Web Connectivity experiment
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// WebConnectivityRequest is a request to run Web Connectivity.
type WebConnectivityRequest struct {
	// Input is the URL to measure using Web Connectivity.
	Input string

	// ReportID is the report ID to use.
	ReportID string

	// TestStartTime is when we started running this test.
	TestStartTime time.Time
}

// WebConnectivityEvent is emitted after we have run
// a measurement using Web Connectivity.
type WebConnectivityEvent struct {
	// Error indicates a fundamental error occurred
	// when running this experiment.
	Error error

	// Measurement is the measurement result.
	Measurement *model.Measurement
}

// webconnectivity performs a measurement using Web Connectivity.
func (s *Session) webconnectivity(ctx context.Context, req *WebConnectivityRequest) {
	runtimex.Assert(req != nil, "passed a nil req")
	measurement, err := s.dowebconnectivity(ctx, req)
	s.maybeEmit(&Event{
		WebConnectivity: &WebConnectivityEvent{
			Error:       err,
			Measurement: measurement,
		},
	})
}

// dowebconnectivity implements webconnectivity.
func (s *Session) dowebconnectivity(
	ctx context.Context, req *WebConnectivityRequest) (*model.Measurement, error) {

	if s.state.IsNone() {
		return nil, ErrNotBootstrapped
	}

	ts := newTickerService(ctx, s)
	defer ts.stop()

	adapter, err := newSessionAdapter(s.state.Unwrap())
	if err != nil {
		return nil, err
	}

	cfg := webconnectivity.Config{}
	runner := webconnectivity.NewExperimentMeasurer(cfg)
	measurement := model.NewMeasurement(
		adapter.location,
		runner.ExperimentName(),
		runner.ExperimentVersion(),
		req.TestStartTime,
		req.ReportID,
		s.state.Unwrap().softwareName,
		s.state.Unwrap().softwareVersion,
		req.Input,
	)
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
		Measurement: measurement,
		Session:     adapter,
	}

	if err := runner.Run(ctx, args); err != nil {
		return nil, err
	}
	return measurement, nil
}
