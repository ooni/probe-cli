package oonimkall

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Task is an asynchronous task running an experiment. It mimics the
// namesake concept initially implemented in Measurement Kit.
//
// # Future directions
//
// Currently Task and Session are two unrelated APIs. As part of
// evolving the APIs with which apps interact with the engine, we
// will modify Task to run in the context of a Session. We will
// do that to save extra lookups and to allow several experiments
// running as subsequent Tasks to reuse the Session connections
// created with the OONI probe services backends.
type Task struct {
	cancel    context.CancelFunc
	isdone    *atomic.Int64
	isstarted chan interface{} // for testing
	isstopped chan interface{} // for testing
	out       chan *event
}

// StartTask starts an asynchronous task. The input argument is a
// serialized JSON conforming to MK v0.10.9's API.
func StartTask(input string) (*Task, error) {
	var settings settings
	if err := json.Unmarshal([]byte(input), &settings); err != nil {
		return nil, err
	}
	const bufsiz = 128 // common case: we don't want runner to block
	ctx, cancel := context.WithCancel(context.Background())
	task := &Task{
		cancel:    cancel,
		isdone:    &atomic.Int64{},
		isstarted: make(chan interface{}),
		isstopped: make(chan interface{}),
		out:       make(chan *event, bufsiz),
	}
	go func() {
		close(task.isstarted)
		emitter := newTaskEmitterUsingChan(task.out)
		r := newRunner(&settings, emitter)
		r.Run(ctx)
		task.out <- nil // signal that we're done w/o closing the channel
		emitter.Close()
		close(task.isstopped)
	}()
	return task, nil
}

// WaitForNextEvent blocks until the next event occurs. The returned
// string is a serialized JSON following MK v0.10.9's API.
func (t *Task) WaitForNextEvent() string {
	const terminated = `{"key":"task_terminated","value":{}}` // like MK
	if t.isdone.Load() != 0 {
		return terminated
	}
	evp := <-t.out
	if evp == nil {
		t.isdone.Add(1)
		return terminated
	}
	data, err := json.Marshal(evp)
	runtimex.PanicOnError(err, "json.Marshal failed")
	return string(data)
}

// IsDone returns true if the task is done.
func (t *Task) IsDone() bool {
	return t.isdone.Load() != 0
}

// Interrupt interrupts the task.
func (t *Task) Interrupt() {
	t.cancel()
}
