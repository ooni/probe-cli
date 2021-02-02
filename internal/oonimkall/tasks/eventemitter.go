package tasks

// EventEmitter emits event on a channel
type EventEmitter struct {
	disabled map[string]bool
	out      chan<- *Event
}

// NewEventEmitter creates a new Emitter
func NewEventEmitter(disabledEvents []string, out chan<- *Event) *EventEmitter {
	ee := &EventEmitter{out: out}
	ee.disabled = make(map[string]bool)
	for _, eventname := range disabledEvents {
		ee.disabled[eventname] = true
	}
	return ee
}

// EmitFailureStartup emits the failureStartup event
func (ee *EventEmitter) EmitFailureStartup(failure string) {
	ee.EmitFailureGeneric(failureStartup, failure)
}

// EmitFailureGeneric emits a failure event
func (ee *EventEmitter) EmitFailureGeneric(name, failure string) {
	ee.Emit(name, EventFailure{Failure: failure})
}

// EmitStatusProgress emits the status.Progress event
func (ee *EventEmitter) EmitStatusProgress(percentage float64, message string) {
	ee.Emit(statusProgress, EventStatusProgress{Message: message, Percentage: percentage})
}

// Emit emits the specified event
func (ee *EventEmitter) Emit(key string, value interface{}) {
	if ee.disabled[key] == true {
		return
	}
	ee.out <- &Event{Key: key, Value: value}
}
