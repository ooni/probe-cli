package main

//
// C API
//

//#include <limits.h>
//#include <stdlib.h>
//
//#include "engine.h"
import "C"

import (
	"encoding/json"
	"log"
	"sync"
	"time"
	"unsafe"
)

//export OONITaskStart
func OONITaskStart(abiVersion, taskName, taskArguments *C.char) C.int {
	return apiSingleton.taskStart(abiVersion, taskName, taskArguments)
}

//export OONITaskWaitForNextEvent
func OONITaskWaitForNextEvent(taskID, timeout C.int) *C.struct_OONIEvent {
	ev := apiSingleton.taskWaitForNextEvent(taskID, time.Duration(timeout)*time.Millisecond)
	if ev == nil {
		// error message already printed
		return nil
	}
	data, err := json.Marshal(ev.value)
	if err != nil {
		log.Printf("OONITaskWaitForNextEvent: cannot serialize JSON: %s", err.Error())
		return nil
	}
	out := (*C.struct_OONIEvent)(C.malloc(C.sizeof_struct_OONIEvent))
	out.Name = C.CString(ev.name)
	out.Value = C.CString(string(data))
	return out
}

//export OONIEventFree
func OONIEventFree(event *C.struct_OONIEvent) {
	if event != nil {
		C.free(unsafe.Pointer(event.Name))
		C.free(unsafe.Pointer(event.Value))
	}
	C.free(unsafe.Pointer(event))
}

//export OONITaskIsDone
func OONITaskIsDone(taskID C.int) C.int {
	return apiSingleton.taskIsDone(taskID)
}

//export OONITaskInterrupt
func OONITaskInterrupt(taskID C.int) {
	apiSingleton.taskInterrupt(taskID)
}

//export OONITaskFree
func OONITaskFree(taskID C.int) {
	apiSingleton.taskFree(taskID)
}

// singleton is the singleton implementing the C API.
var apiSingleton = newAPI()

// newAPI creates a new instance of api.
func newAPI() *api {
	return &api{
		mockableInsertTask: insertTask,
		mockableStartTask:  startTask,
		mu:                 sync.Mutex{},
		nextid:             0,
		tasks:              map[C.int]taskAPI{},
	}
}

// api implements the C API.
type api struct {
	// mockableInsertTask calls insertTask indirectly, thus allowing for testing.
	mockableInsertTask func(api *api, tp taskAPI) C.int

	// mockableStartTask calls startTask indirectly, thus allowing for testing.
	mockableStartTask func(name string, args []byte) taskAPI

	// mu provides mutual exclusion when accessing the C API.
	mu sync.Mutex

	// nextid is the next task's ID.
	nextid C.int

	// tasks tracks tasks that have been started.
	tasks map[C.int]taskAPI
}

// invalidTaskID is the canonical representation of an invalid taskID. Any negative value
// represents an invalid task, but it's good to have a canonical representation.
const invalidTaskID = -1

// taskStart implements OONITaskStart.
func (a *api) taskStart(abiVersion, taskName, taskArguments *C.char) C.int {
	if abiVersion == nil {
		log.Printf("OONITaskStart: abiVersion cannot be NULL")
		return invalidTaskID
	}
	if v := C.GoString(abiVersion); v != ABIVersion {
		log.Printf("OONITaskStart: ABI mismatch: expected %s, got %s", ABIVersion, v)
		return invalidTaskID
	}
	if taskName == nil {
		log.Printf("OONITaskStart: taskName cannot be NULL")
		return invalidTaskID
	}
	if taskArguments == nil {
		log.Printf("OONITaskStart: taskArguments cannot be NULL")
		return invalidTaskID
	}
	tp := a.mockableStartTask(C.GoString(taskName), []byte(C.GoString(taskArguments)))
	if tp == nil {
		log.Print("OONITaskStart: startTask returned NULL")
		return invalidTaskID
	}
	taskID := a.mockableInsertTask(a, tp)
	if taskID < 0 {
		log.Print("OONITaskStart: cannot find a free slot for this task")
		tp.free()
		return invalidTaskID
	}
	return taskID
}

// insertTask inserts a task inside the task list and returns
// its identifier. A negative return value indicates we couldn't
// find room to insert this task (_very_ unlikely).
func insertTask(a *api, tp taskAPI) C.int {
	a.mu.Lock()
	defer a.mu.Unlock()
	orig := a.nextid
	for {
		if a.tasks[a.nextid] == nil {
			task := a.nextid
			incrementNextIDLocked(a)
			a.tasks[task] = tp
			return task
		}
		incrementNextIDLocked(a)
		if orig == a.nextid {
			return invalidTaskID
		}
	}
}

// incrementNextIDLocked increments the next task ID wrapping the
// value back to zero when we read C.INT_MAX. This function MUST be
// called while holding a.mu, as its name implies.
func incrementNextIDLocked(a *api) {
	if a.nextid > C.INT_MAX-1 {
		a.nextid = 0
	} else {
		a.nextid++
	}
}

// taskWaitForNextEvent implements OONITaskWaitForNextEvent.
func (a *api) taskWaitForNextEvent(taskID C.int, timeout time.Duration) *taskEvent {
	a.mu.Lock()
	tp := a.tasks[taskID]
	a.mu.Unlock()
	if tp == nil {
		log.Printf("OONITaskWaitForNextEvent: task %d does not exist", taskID)
		return nil
	}
	return tp.waitForNextEvent(timeout)
}

// taskIsDone implements OONITaskIsDone.
func (a *api) taskIsDone(taskID C.int) C.int {
	a.mu.Lock()
	tp := a.tasks[taskID]
	a.mu.Unlock()
	if tp == nil {
		log.Printf("OONITaskIsDone: task %d does not exist", taskID)
		return 1 // a nonexistent task is always done
	}
	out := C.int(0)
	if tp.isDone() {
		out++ // nonzero if done
	}
	return out
}

// taskInterrupt implements OONITaskInterrupt.
func (a *api) taskInterrupt(taskID C.int) {
	a.mu.Lock()
	tp := a.tasks[taskID]
	a.mu.Unlock()
	if tp == nil {
		log.Printf("OONITaskInterrupt: task %d does not exist", taskID)
		return
	}
	tp.interrupt()
}

// taskFree implements OONITaskFree.
func (a *api) taskFree(taskID C.int) {
	a.mu.Lock()
	tp := a.tasks[taskID]
	delete(a.tasks, taskID) // this forgets the ID->task binding
	a.mu.Unlock()
	if tp == nil {
		// No need to print a warning message here. We want logging
		// idempotence because may may end up killing a task more
		// than once for robustness and we don't want our robustness
		// aims to spew suspicious messages at our users.
		return
	}
	tp.free()
}

func main() {}
