package oonimkall

import "sync"

// taskEmitterUsingChan is a task emitter using a channel.
type taskEmitterUsingChan struct {
	// eof indicates we should not emit anymore.
	eof chan interface{}

	// once ensures we close the eof channel just once.
	once sync.Once

	// out is the possibly buffered channel where to emit events.
	out chan<- *event
}

// ensure that taskEmitterUsingChan is a taskEmitter.
var _ taskEmitterCloser = &taskEmitterUsingChan{}

// newTaskEmitterUsingChan creates a taskEmitterUsingChan.
func newTaskEmitterUsingChan(out chan<- *event) *taskEmitterUsingChan {
	return &taskEmitterUsingChan{
		eof:  make(chan interface{}),
		once: sync.Once{},
		out:  out,
	}
}

// Emit implements taskEmitter.Emit.
func (ee *taskEmitterUsingChan) Emit(key string, value interface{}) {
	// Prevent this goroutine from blocking on `ee.out` if the caller
	// has already told us it's not going to accept more events.
	select {
	case ee.out <- &event{Key: key, Value: value}:
	case <-ee.eof:
	}
}

// Close implements taskEmitterCloser.Closer.
func (ee *taskEmitterUsingChan) Close() error {
	ee.once.Do(func() { close(ee.eof) })
	return nil
}

// taskEmitterWrapper is a convenient wrapper for taskEmitter.
type taskEmitterWrapper struct {
	taskEmitter
}

// EmitFailureStartup emits the failureStartup event
func (ee *taskEmitterWrapper) EmitFailureStartup(failure string) {
	ee.EmitFailureGeneric(failureStartup, failure)
}

// EmitFailureGeneric emits a failure event
func (ee *taskEmitterWrapper) EmitFailureGeneric(name, failure string) {
	ee.Emit(name, eventFailure{Failure: failure})
}

// EmitStatusProgress emits the status.Progress event
func (ee *taskEmitterWrapper) EmitStatusProgress(percentage float64, message string) {
	ee.Emit(statusProgress, eventStatusProgress{Message: message, Percentage: percentage})
}
