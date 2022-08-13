package main

//
// Model
//

import (
	"context"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
)

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
	maybeEmitEvent(name string, value protoreflect.ProtoMessage)
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
	// - args contains unparsed, task-specific arguments.
	main(ctx context.Context, emitter taskMaybeEmitter, args []byte)
}

// goMessage the internal representation of OONIMessage.
type goMessage struct {
	// key is the event key.
	key string

	// value is the value of the event.
	value protoreflect.ProtoMessage
}

// taskAPI implements the OONI engine C API functions. We use this interface
// to enable easier testing of the code that manages the tasks lifecycle.
type taskAPI interface {
	// waitForNextEvent implements OONITaskWaitForNextEvent.
	waitForNextEvent(timeout time.Duration) *goMessage

	// isDone implements OONITaskIsDone.
	isDone() bool

	// interrupt implements OONITaskInterrupt.
	interrupt()

	// free implements OONITaskFree.
	free()
}

// NewFailureString maps an error to a failure string using the
// empty string to represent the absence of errors. This representation
// is the one we use to deliver failures to C API clients.
func newFailureString(err error) (out string) {
	if err != nil {
		out = err.Error()
	}
	return
}

// taskRegistry maps each task name to its implementation.
var taskRegistry = map[string]taskRunner{}
