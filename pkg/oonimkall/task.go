// Package oonimkall implements APIs used by OONI mobile apps. We
// expose these APIs to mobile apps using gomobile.
//
// We expose two APIs: the task API, which is derived from the
// API originally exposed by Measurement Kit, and the session API,
// which is a Go API that mobile apps can use via `gomobile`.
//
// This package is named oonimkall because it contains a partial
// reimplementation of the mkall API implemented by Measurement Kit
// in, e.g., https://github.com/measurement-kit/mkall-ios.
//
// Task API
//
// The basic tenet of the task API is that you define an experiment
// task you wanna run using a JSON, then you start a task for it, and
// you receive events as serialized JSONs. In addition to this
// functionality, we also include extra APIs used by OONI mobile.
//
// The task API was first defined in Measurement Kit v0.9.0. In this
// context, it was called "the FFI API". The API we expose here is not
// strictly an FFI API, but is close enough for the purpose of using
// OONI from Android and iOS. See https://git.io/Jv4Rv
// (measurement-kit/measurement-kit@v0.10.9) for a comprehensive
// description of MK's FFI API.
//
// See also https://github.com/ooni/probe-engine/pull/347 for the
// design document describing the task API.
//
// See also https://github.com/ooni/probe-cli/v3/internal/engine/blob/master/DESIGN.md,
// which explains why we implemented the oonimkall API.
//
// Session API
//
// The Session API is a Go API that can be exported to mobile apps
// using the gomobile tool. The latest design document for this API is
// at https://github.com/ooni/probe-engine/pull/954.
//
// The basic tenet of the session API is that you create an instance
// of `Session` and use it to perform the operations you need.
package oonimkall

import (
	"context"
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Task is an asynchronous task running an experiment. It mimics the
// namesake concept initially implemented in Measurement Kit.
//
// Future directions
//
// Currently Task and Session are two unrelated APIs. As part of
// evolving the APIs with which apps interact with the engine, we
// will modify Task to run in the context of a Session. We will
// do that to save extra lookups and to allow several experiments
// running as subsequent Tasks to reuse the Session connections
// created with the OONI probe services backends.
type Task struct {
	cancel    context.CancelFunc
	isdone    *atomicx.Int64
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
		isdone:    &atomicx.Int64{},
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
