package session

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// ProgressEvent indicates the progress in a task completion.
type ProgressEvent struct {
	// Timestamp is the timestamp.
	Timestamp time.Time

	// Completion is a number between 0 and 1.
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
	pb.emit(completion, message)
}

// emit emits a progress event.
func (pb *progressBar) emit(completion float64, message string) {
	ev := &Event{
		Progress: &ProgressEvent{
			Timestamp:  time.Now(),
			Completion: completion,
			Message:    message,
		},
	}
	pb.session.emit(ev)
}
