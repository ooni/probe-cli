package miniengine

//
// Measurement callbacks
//

import "github.com/ooni/probe-cli/v3/internal/model"

// callbacks implements [model.ExperimentCallbacks]
type callbacks struct {
	emitter chan<- *Event
}

var _ model.ExperimentCallbacks = &callbacks{}

// OnProgress implements model.ExperimentCallbacks
func (c *callbacks) OnProgress(progress float64, message string) {
	event := &Event{
		EventType: EventTypeProgress,
		Message:   message,
		Progress:  progress,
	}
	select {
	case c.emitter <- event:
	default:
	}
}
