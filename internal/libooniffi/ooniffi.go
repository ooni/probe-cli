package main

import (
	//#include "ooniffi.h"
	//
	//#include <stdint.h>
	//#include <stdlib.h>
	//
	//struct ooniffi_task_ {
	//    int64_t Handle;
	//};
	//
	// struct ooniffi_event_ {
	//    char *String;
	//};
	"C"
	"sync"
	"unsafe"

	"github.com/ooni/probe-cli/v3/internal/oonimkall"
)

var (
	idx C.int64_t
	m   = make(map[C.int64_t]*oonimkall.Task)
	mu  sync.Mutex
)

func cstring(s string) *C.char {
	return C.CString(s)
}

func freestring(s *C.char) {
	C.free(unsafe.Pointer(s))
}

func gostring(s *C.char) string {
	return C.GoString(s)
}

const maxIdx = C.INT64_MAX

//export ooniffi_task_start_
func ooniffi_task_start_(settings *C.char) *C.ooniffi_task_t {
	if settings == nil {
		return nil
	}
	tp, err := oonimkall.StartTask(gostring(settings))
	if err != nil {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	// TODO(bassosimone): the following if is basic protection against
	// undefined behaviour, i.e., the counter wrapping around. A much
	// better strategy would probably be to restart from 0. However it's
	// also unclear if any device could run that many tests, so...
	if idx >= maxIdx {
		return nil
	}
	handle := idx
	idx++
	m[handle] = tp
	task := (*C.ooniffi_task_t)(C.malloc(C.sizeof_ooniffi_task_t))
	task.Handle = handle
	return task
}

func setmaxidx() C.int64_t {
	o := idx
	idx = maxIdx
	return o
}

func restoreidx(v C.int64_t) {
	idx = v
}

//export ooniffi_task_wait_for_next_event
func ooniffi_task_wait_for_next_event(task *C.ooniffi_task_t) (event *C.ooniffi_event_t) {
	if task != nil {
		mu.Lock()
		tp := m[task.Handle]
		mu.Unlock()
		if tp != nil {
			event = (*C.ooniffi_event_t)(C.malloc(C.sizeof_ooniffi_event_t))
			event.String = cstring(tp.WaitForNextEvent())
		}
	}
	return
}

//export ooniffi_task_is_done
func ooniffi_task_is_done(task *C.ooniffi_task_t) C.int {
	var isdone C.int = 1
	if task != nil {
		mu.Lock()
		if tp := m[task.Handle]; tp != nil && !tp.IsDone() {
			isdone = 0
		}
		mu.Unlock()
	}
	return isdone
}

//export ooniffi_task_interrupt
func ooniffi_task_interrupt(task *C.ooniffi_task_t) {
	if task != nil {
		mu.Lock()
		if tp := m[task.Handle]; tp != nil {
			tp.Interrupt()
		}
		mu.Unlock()
	}
}

//export ooniffi_event_serialization_
func ooniffi_event_serialization_(event *C.ooniffi_event_t) (s *C.char) {
	if event != nil {
		s = event.String
	}
	return
}

//export ooniffi_event_destroy
func ooniffi_event_destroy(event *C.ooniffi_event_t) {
	if event != nil {
		C.free(unsafe.Pointer(event.String))
		C.free(unsafe.Pointer(event))
	}
}

//export ooniffi_task_destroy
func ooniffi_task_destroy(task *C.ooniffi_task_t) {
	if task != nil {
		mu.Lock()
		tp := m[task.Handle]
		delete(m, task.Handle)
		mu.Unlock()
		C.free(unsafe.Pointer(task))
		if tp != nil { // drain task if needed
			tp.Interrupt()
			go func() {
				for !tp.IsDone() {
					tp.WaitForNextEvent()
				}
			}()
		}
	}
}

func main() {}
