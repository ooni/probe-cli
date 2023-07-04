package miniengine

//
// The "measure" task
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/context"
)

// MeasurementResult contains the results of [Session.Measure]
type MeasurementResult struct {
	// KibiBytesReceived contains the KiB we received
	KibiBytesReceived float64 `json:"kibi_bytes_received"`

	// KibiBytesSent contains the KiB we sent
	KibiBytesSent float64 `json:"kibi_bytes_sent"`

	// Measurement is the generated [model.Measurement]
	Measurement *model.Measurement `json:"measurement"`

	// Summary is the corresponding summary.
	Summary any `json:"summary"`
}

// Measure performs a measurement using the given experiment and input.
func (s *Session) Measure(
	ctx context.Context,
	name string,
	options map[string]any,
	input string,
) *Task[*MeasurementResult] {
	task := &Task[*MeasurementResult]{
		done:    make(chan any),
		events:  s.emitter,
		failure: nil,
		result:  nil,
	}
	go s.measureAsync(ctx, name, options, input, task)
	return task
}

// measureAsync runs the measurement in a background goroutine.
func (s *Session) measureAsync(
	ctx context.Context,
	name string,
	options map[string]any,
	input string,
	task *Task[*MeasurementResult],
) {
	// synchronize with Task.Result
	defer close(task.done)

	// lock and access the underlying session
	s.mu.Lock()
	defer s.mu.Unlock()

	// handle the case where we did not bootstrap
	if s.state.IsNone() {
		task.failure = ErrNoBootstrap
		return
	}
	sess := s.state.Unwrap().sess

	// create a [model.ExperimentBuilder]
	builder, err := sess.NewExperimentBuilder(name)
	if err != nil {
		task.failure = err
		return
	}

	// set the proper callbacks for the experiment
	callbacks := &callbacks{s.emitter}
	builder.SetCallbacks(callbacks)

	// set the proper options for the experiment
	if err := builder.SetOptionsAny(options); err != nil {
		task.failure = err
		return
	}

	// create an experiment instance
	exp := builder.NewExperiment()

	// perform the measurement
	meas, err := exp.MeasureWithContext(ctx, input)
	if err != nil {
		task.failure = err
		return
	}

	// obtain the summary
	summary, err := exp.GetSummaryKeys(meas)
	if err != nil {
		task.failure = err
		return
	}

	// pass response to the caller
	task.result = &MeasurementResult{
		KibiBytesReceived: exp.KibiBytesReceived(),
		KibiBytesSent:     exp.KibiBytesSent(),
		Measurement:       meas,
		Summary:           summary,
	}
}
