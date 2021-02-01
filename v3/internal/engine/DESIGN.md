# Replacing Measurement Kit

| Author       | Simone Basso |
|--------------|--------------|
| Last-Updated | 2020-07-09   |
| Status       | historical   |

*Abstract* We describe our plan of replacing Measurement Kit for OONI
Probe Android and iOS (in particular) and (also) the CLI.

## Introduction

We want to write experiments in Go. This reduces our burden
compared to writing them using C/C++ code.

Go consumers of probe-engine shall directly use its Go API. We
will discuss the Go API in a future revision of this spec.

For mobile apps, we want to replace these MK APIs:

- [measurement-kit/android-libs](https://github.com/measurement-kit/android-libs)

- [measurement-kit/mkall-ios](https://github.com/measurement-kit/mkall-ios)

We also want consumers of [measurement-kit's FFI API](https://git.io/Jv4Rv)
to be able to replace measurement-kit with probe-engine.

## APIs to replace

### Mobile APIs

We define a Go API that `gomobile` binds to a Java/ObjectiveC
API that is close enough to the MK's mobile APIs.

### FFI API

We define a CGO API such that `go build -buildmode=c-shared`
yields an API reasonably close to MK's FFI API.

## Running experiments

It seems the generic API for enabling running experiments both on
mobile devices and for FFI consumers is like:

```Go
type Task struct{ ... }
  func StartTask(input string) (*Task, error)
  func (t *Task) Interrupt()
  func (t *Task) IsDone() bool
  func (t *Task) WaitForNextEvent() string
```

This should be enough to generate a suitable mobile API when
using the `gomobile` Go subcommand.

We can likewise generate a FFI API as follows:

```Go
package main

import (
  "C"
  "sync"

  "github.com/ooni/probe-engine/oonimkall"
)

var (
  idx int64 = 1
  m         = make(map[int64]*oonimkall.Task)
  mu        sync.Mutex
)

//export ooni_task_start
func ooni_task_start(settings string) int64 {
  tp, err := oonimkall.StartTask(settings)
  if err != nil {
    return 0
  }
  mu.Lock()
  handle := idx
  idx++
  m[handle] = tp
  mu.Unlock()
  return handle
}

//export ooni_task_interrupt
func ooni_task_interrupt(handle int64) {
  mu.Lock()
  if tp := m[handle]; tp != nil {
    tp.Interrupt()
  }
  mu.Unlock()
}

//export ooni_task_is_done
func ooni_task_is_done(handle int64) bool {
  isdone := true
  mu.Lock()
  if tp := m[handle]; tp != nil {
    isdone = tp.IsDone()
  }
  mu.Unlock()
  return isdone
}

//export ooni_task_wait_for_next_event
func ooni_task_wait_for_next_event(handle int64) (event string) {
  mu.Lock()
  tp := m[handle]
  mu.Unlock()
  if tp != nil {
    event = tp.WaitForNextEvent()
  }
  return
}

func main() {}
```

This is close enough to [measurement-kit's FFI API](https://git.io/Jv4Rv) that
a few lines of C allow to implement an ABI compatible replacement.

## Other APIs of interest

We currently don't have plans for replacing other MK APIs. We will introduce
new APIs specifically tailored for our OONI needs, but they will be out of
scope with respect to the main goal of this design document.

## History

[The initial version of this design document](
https://github.com/measurement-kit/engine/blob/master/DESIGN.md)
lived in the measurement-kit namespace at GitHub. It discussed
a bunch of broad, extra topics, e.g., code bloat that are not
discussed in this document. More details regarding the migration
from MK to probe-engine are at [measurement-kit/measurement-kit#1913](
https://github.com/measurement-kit/measurement-kit/issues/1913).
