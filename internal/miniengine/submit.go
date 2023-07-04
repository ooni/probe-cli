package miniengine

//
// The "submit" task
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/context"
)

// Submit submits a [model.Measurement] to the OONI backend. You MUST initialize
// the measurement's report ID. You can find the report ID for each experiment
// in the results of the check-in API.
func (s *Session) Submit(ctx context.Context, meas *model.Measurement) *Task[Void] {
	task := &Task[Void]{
		done:    make(chan any),
		events:  s.emitter,
		failure: nil,
		result:  Void{},
	}
	go s.submitAsync(ctx, meas, task)
	return task
}

// submitAsync submits the measurement in a background goroutine.
func (s *Session) submitAsync(ctx context.Context, meas *model.Measurement, task *Task[Void]) {
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
	state := s.state.Unwrap()

	// submit the measurement to the backend
	task.failure = state.psc.SubmitMeasurement(ctx, meas)
}
