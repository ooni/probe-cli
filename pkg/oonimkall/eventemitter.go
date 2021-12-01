package oonimkall

// eventEmitter emits event on a channel
type eventEmitter struct {
	disabled map[string]bool
	eof      <-chan interface{}
	out      chan<- *event
}

// newEventEmitter creates a new Emitter
func newEventEmitter(disabledEvents []string, out chan<- *event,
	eof <-chan interface{}) *eventEmitter {
	ee := &eventEmitter{eof: eof, out: out}
	ee.disabled = make(map[string]bool)
	for _, eventname := range disabledEvents {
		ee.disabled[eventname] = true
	}
	return ee
}

// EmitFailureStartup emits the failureStartup event
func (ee *eventEmitter) EmitFailureStartup(failure string) {
	ee.EmitFailureGeneric(failureStartup, failure)
}

// EmitFailureGeneric emits a failure event
func (ee *eventEmitter) EmitFailureGeneric(name, failure string) {
	ee.Emit(name, eventFailure{Failure: failure})
}

// EmitStatusProgress emits the status.Progress event
func (ee *eventEmitter) EmitStatusProgress(percentage float64, message string) {
	ee.Emit(statusProgress, eventStatusProgress{Message: message, Percentage: percentage})
}

// Emit emits the specified event
func (ee *eventEmitter) Emit(key string, value interface{}) {
	if ee.disabled[key] {
		return
	}
	// Prevent this goroutine from blocking on `ee.out` if the caller
	// has already told us it's not going to accept more events.
	select {
	case ee.out <- &event{Key: key, Value: value}:
	case <-ee.eof:
	}
}
