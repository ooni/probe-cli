package nettests

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/miniengine"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// taskLogger is the logger used for logging tasks.
var taskLogger = log.WithFields(log.Fields{
	"type": "engine",
})

// runningTask is a [miniengine.Task] that is still running.
type runningTask interface {
	Done() <-chan any
	Events() <-chan *miniengine.Event
}

// TODO(bassosimone): we need to set the verbosity

// processTaskEvent processes an event emitted by a runingTask.
func processTaskEvent(callbacks model.ExperimentCallbacks, ev *miniengine.Event) {
	switch ev.EventType {
	case miniengine.EventTypeDebug:
		taskLogger.Debug(ev.Message)
	case miniengine.EventTypeInfo:
		taskLogger.Info(ev.Message)
	case miniengine.EventTypeProgress:
		callbacks.OnProgress(ev.Progress, ev.Message)
	case miniengine.EventTypeWarning:
		taskLogger.Warn(ev.Message)
	default:
		taskLogger.Warnf("UNHANDLED EVENT: %+v", ev)
	}
}

// awaitTask waits for the given runningTask to terminate.
func awaitTask(task runningTask, callbacks model.ExperimentCallbacks) {
	for {
		select {
		case <-task.Done():
			for {
				select {
				case ev := <-task.Events():
					processTaskEvent(callbacks, ev)
				default:
					return
				}
			}
		case ev := <-task.Events():
			processTaskEvent(callbacks, ev)
		}
	}
}
