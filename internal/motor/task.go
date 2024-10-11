package motor

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"
)

var (
	errInvalidRequest = errors.New("input request has no valid task name")
)

// startTask starts a given task.
func StartTask(req *Request) TaskAPI {
	ctx, cancel := context.WithCancel(context.Background())
	tp := &taskState{
		cancel:  cancel,
		done:    &atomic.Int64{},
		events:  make(chan *Response, taskEventsBuffer),
		result:  make(chan *Response, 1),
		stopped: make(chan any),
	}
	go tp.main(ctx, req)
	return tp
}

// task implements taskAPI.
type taskState struct {
	// cancel cancels the context used by this task.
	cancel context.CancelFunc

	// done indicates that this task is done.
	done *atomic.Int64

	// events is the channel where we emit task events.
	events chan *Response

	// result is the channel where we emit the final result.
	result chan *Response

	// stopped indicates that the task is done.
	stopped chan any
}

var _ TaskAPI = &taskState{}

// WaitForNextEvent implements TaskAPI.WaitForNextEvent.
func (tp *taskState) WaitForNextEvent(timeout time.Duration) *Response {
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

// Result implements TaskAPI.Result
func (tp *taskState) Result() *Response {
	return <-tp.result
}

// contextForWaitForNextEvent returns the suitable context
// for making the waitForNextEvent function time bounded.
func contextForWaitForNextEvent(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if timeout < 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

// IsDone implements TaskAPI.IsDone.
func (tp *taskState) IsDone() bool {
	return tp.done.Load() > 0
}

// Interrupt implements TaskAPI.Interrupt.
func (tp *taskState) Interrupt() {
	tp.cancel()
}

// main is the main function of the task.
func (tp *taskState) main(ctx context.Context, req *Request) {
	defer close(tp.stopped) // synchronize with caller
	taskName := req.Name
	resp := &Response{}
	runner := taskRegistry[taskName]
	if runner == nil {
		log.Printf("OONITaskStart: unknown task name: %s", taskName)
		resp.Error = errInvalidRequest.Error()
		tp.result <- resp
		return
	}
	emitter := &taskChanEmitter{
		out: tp.events,
	}
	runner.main(ctx, emitter, req, resp)
	tp.result <- resp // emit response to result channel
}
