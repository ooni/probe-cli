package main

import (
	"context"
	"time"
)

// request is the OONI request containing task-specific arguments.
type request struct {
	NewSession    newSessionOptions    `json:",omitempty"`
	Geolocate     geolocateOptions     `json:",omitempty"`
	DeleteSession deleteSessionOptions `json:",omitempty"`
	Test          testOptions          `json:",omitempty"`
}

// response is the OONI response to serialize before sending.
type response struct {
	NewSession    newSessionResponse    `json:",omitempty"`
	Geolocate     geolocateResponse     `json:",omitempty"`
	Logger        logResponse           `json:",omitempty"`
	DeleteSession deleteSessionResponse `json:",omitempty"`
	Test          testResponse          `json:",omitempty"`
	Error         string                `json:",omitempty"`
}

// taskEventsBuffer is the buffer used for the task's event chan, which
// should guarantee enough buffering when the application is slow.
const taskEventsBuffer = 1024

// taskMaybeEmitter emits events, if possible. We use a buffered
// channel with a large buffer for collecting task events. We expect
// the application to always be able to drain the channel timely. Yet,
// if that's not the case, it is fine to discard events. This data
// type implement such a discard-if-reader is slow behaviour.
type taskMaybeEmitter interface {
	// maybeEmitEvent emits an event if there's available buffer in the
	// output channel and otherwise discards the event.
	maybeEmitEvent(resp *response)
}

// taskRunner runs a given task. Any task that you can run from
// the application must implement this interface.
type taskRunner interface {
	// Main runs the task to completion.
	//
	// Arguments:
	//
	// - ctx is the context for deadline/cancellation/timeout;
	//
	// - emitter is the emitter to emit events;
	//
	// - req is the parsed request containing task specific arguments.
	//
	// - resp is the response to emit after the task is complete. Note that
	//   this an implicit response and only indicates the final response of the task.
	main(ctx context.Context, emitter taskMaybeEmitter, req *request, resp *response)
}

// taskAPI implements the OONI engine C API functions. We use this interface
// to enable easier testing of the code that manages the tasks lifecycle.
type taskAPI interface {
	// waitForNextEvent implements OONITaskWaitForNextEvent.
	waitForNextEvent(timeout time.Duration) *response

	// isDone implements OONITaskIsDone.
	isDone() bool

	// interrupt implements OONITaskInterrupt.
	interrupt()

	// free implements OONITaskFree
	free()
}

// taskRegistry maps each task name to its implementation.
var taskRegistry = map[string]taskRunner{}
