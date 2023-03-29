package session

//
// Progress bar for experiments that manage their own progress
// bar such as DASH, NDT, HIRL, HHFM.
//

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// ProgressEvent indicates the progress in a task completion
// for tasks that manage their own progress bar.
type ProgressEvent struct {
	// Timestamp is the timestamp.
	Timestamp time.Time

	// Completion is a number between 0 and 1 indicating
	// how close we are to completion.
	Completion float64

	// Message is the message.
	Message string
}

// progressBar emits progress.
type progressBar struct {
	session *Session
}

var _ model.ExperimentCallbacks = &progressBar{}

// OnProgress implements model.ExperimentCallbacks
func (pb *progressBar) OnProgress(completion float64, message string) {
	pb.maybeEmit(completion, message)
}

// emit emits a progress event unless the output channel is full.
func (pb *progressBar) maybeEmit(completion float64, message string) {
	ev := &Event{
		Progress: &ProgressEvent{
			Timestamp:  time.Now(),
			Completion: completion,
			Message:    message,
		},
	}
	pb.session.maybeEmit(ev)
}
