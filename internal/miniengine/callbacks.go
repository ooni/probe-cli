package miniengine

//
// Measurement callbacks
//

import "github.com/ooni/probe-cli/v3/internal/model"

// callbacks implements [model.ExperimentCallbacks] and emits
// the callbacks events using the given channel.
type callbacks struct {
	// emitter is the channel where to emit events.
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
	// Implementation note: it's fine to lose interim events
	select {
	case c.emitter <- event:
	default:
	}
}
