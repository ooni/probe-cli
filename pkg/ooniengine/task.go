package main

//
// Task implementation
//

import (
	"context"
	"log"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

// startTask starts a given task.
func startTask(name string, args []byte) taskAPI {
	ctx, cancel := context.WithCancel(context.Background())
	tp := &taskState{
		cancel:  cancel,
		done:    &atomicx.Int64{},
		events:  make(chan *goMessage, taskEventsBuffer),
		stopped: make(chan any),
	}
	go tp.main(ctx, name, args)
	return tp
}

// task implements taskAPI.
type taskState struct {
	// cancel cancels the context used by this task.
	cancel context.CancelFunc

	// done indicates that this task is done.
	done *atomicx.Int64

	// events is the channel where we emit task events.
	events chan *goMessage

	// stopped indicates that the task is done.
	stopped chan any
}

var _ taskAPI = &taskState{}

// waitForNextEvent implements taskAPI.waitForNextEvent.
func (tp *taskState) waitForNextEvent(timeout time.Duration) *goMessage {
	// Implementation note: we don't need to log any of these nil-returning conditions
	// as they are not exceptional, rather they're part of normal usage.
	ctx, cancel := contextForWaitForNextEvent(timeout)
	defer cancel()
	select {
	case <-ctx.Done():
		return nil // timeout while blocking for reading
	case ev := <-tp.events:
		return ev // ordinary chan reading
	case <-tp.stopped:
		select {
		case ev := <-tp.events:
			return ev // still draining the chan
		default:
			tp.done.Add(1) // fully drained so we can flip "done" now
			return nil
		}
	}
}

// contextForWaitForNextEvent returns the suitable context
// for making the waitForNextEvent function time bounded.
func contextForWaitForNextEvent(timeo time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if timeo < 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeo)
}

// isDone implements taskAPI.isDone.
func (tp *taskState) isDone() bool {
	return tp.done.Load() > 0
}

// interrupt implements taskAPI.interrupt.
func (tp *taskState) interrupt() {
	tp.cancel()
}

// free implements taskAPI.free.
func (tp *taskState) free() {
	tp.interrupt() // interrupt immediately
	for !tp.isDone() {
		const blockForever = -1
		_ = tp.waitForNextEvent(blockForever) // drain until done
	}
}

// main is the main function of the task.
func (tp *taskState) main(ctx context.Context, name string, args []byte) {
	defer close(tp.stopped) // synchronize with caller
	runner := taskRegistry[name]
	if runner == nil {
		log.Printf("OONITaskStart: unknown task name: %s", name)
		return
	}
	emitter := &taskChanEmitter{
		out: tp.events,
	}
	runner.main(ctx, emitter, args)
}
