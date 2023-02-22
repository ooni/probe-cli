package main

//
// Emitter
//

// taskEmitter implements taskMaybeEmitter.
type taskChanEmitter struct {
	// out is the channel where we emit events.
	out chan *goMessage
}

var _ taskMaybeEmitter = &taskChanEmitter{}

// maybeEmitEvent implements taskMaybeEmitter.maybeEmitEvent.
func (e *taskChanEmitter) maybeEmitEvent(name string, value event) {
	ev := &goMessage{
		key:   name,
		value: value,
	}
	select {
	case e.out <- ev:
	default: // buffer full, discard this event
	}
}
