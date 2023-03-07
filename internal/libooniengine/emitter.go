package main

//
// Emitter
//

// taskEmitter implements taskMaybeEmitter.
type taskChanEmitter struct {
	// out is the channel where we emit events.
	out chan *response
}

var _ taskMaybeEmitter = &taskChanEmitter{}

// maybeEmitEvent implements taskMaybeEmitter.maybeEmitEvent.
func (e *taskChanEmitter) maybeEmitEvent(resp *response) {
	select {
	case e.out <- resp:
	default: // buffer full, discard this event
	}
}
