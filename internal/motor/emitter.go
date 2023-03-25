package motor

//
// Emitter
//

// taskEmitter implements taskMaybeEmitter.
type taskChanEmitter struct {
	// out is the channel where we emit events.
	out chan *Response
}

var _ taskMaybeEmitter = &taskChanEmitter{}

// maybeEmitEvent implements taskMaybeEmitter.maybeEmitEvent.
func (e *taskChanEmitter) maybeEmitEvent(resp *Response) {
	select {
	case e.out <- resp:
	default: // buffer full, discard this event
	}
}
