package miniengine

//
// Task
//

import "golang.org/x/net/context"

// Task is a long running operation that emits [Event] while it is running and
// produces a given Result. The zero value of this struct is invalid; you cannot
// create a valid [Task] outside of this package.
type Task[Result any] struct {
	// done is closed when the [Task] is done.
	done chan any

	// events is where the [Task] emits [Event].
	events chan *Event

	// failure is the [Task] failure or nil.
	failure error

	// result is the [Task] result (zero on failure).
	result Result
}

// TaskRunner runs the main function that produces a [Task] result.
type TaskRunner[Result any] interface {
	// Main is the [Task] main function.
	Main(ctx context.Context) (Result, error)
}

// Done returns a channel closed when the [Task] is done.
func (t *Task[Result]) Done() <-chan any {
	return t.done
}

// Events returns a channel where a running [Task] emits [Event].
func (t *Task[Result]) Events() <-chan *Event {
	return t.events
}

// Result returns the [Task] result (if the task succeded) or the error that
// occurred (in case of failure). This method blocks until the channel returned
// by the [Task.Done] method has been closed.
func (t *Task[Result]) Result() (Result, error) {
	<-t.done // synchronize with TaskRunner.Main
	return t.result, t.failure
}
