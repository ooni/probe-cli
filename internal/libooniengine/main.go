package main

//
// C API
//

//#include <stdlib.h>
//
//#include "engine.h"
import "C"

import (
	"encoding/json"
	"log"
	"time"
	"unsafe"

	"github.com/ooni/probe-cli/v3/internal/motor"
	"github.com/ooni/probe-cli/v3/internal/version"
)

const (
	// invalidTaskHandle represents the invalid task handle.
	invalidTaskHandle = 0
)

// parse converts a JSON request string to the concrete Go type.
func parse(req *C.char) (*motor.Request, error) {
	out := &motor.Request{}
	if err := json.Unmarshal([]byte(C.GoString(req)), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// serialize serializes a OONI response to a JSON string accessible to C code.
func serialize(resp *motor.Response) *C.char {
	if resp == nil {
		return nil
	}
	out, err := json.Marshal(resp)
	if err != nil {
		log.Printf("serializeMessage: cannot serialize message: %s", err.Error())
		return nil
	}
	return C.CString(string(out))
}

//export OONIEngineVersion
func OONIEngineVersion() *C.char {
	return C.CString(version.Version)
}

//export OONIEngineFreeMemory
func OONIEngineFreeMemory(ptr *C.void) {
	C.free(unsafe.Pointer(ptr))
}

//export OONIEngineCall
func OONIEngineCall(req *C.char) C.OONITask {
	r, err := parse(req)
	if err != nil {
		log.Printf("OONIEngineCall: %s", err.Error())
		return invalidTaskHandle
	}
	tp := motor.StartTask(r)
	if tp == nil {
		log.Printf("OONITaskStart: startTask returned NULL")
		return invalidTaskHandle
	}
	handle, err := handler.newHandle(tp)
	if err != nil {
		log.Printf("OONITaskStart: %s", err.Error())
		return invalidTaskHandle
	}
	return C.OONITask(handle)
}

//export OONIEngineWaitForNextEvent
func OONIEngineWaitForNextEvent(task C.OONITask, timeout C.int32_t) *C.char {
	tp := handler.getTaskHandle(task)
	if tp == nil {
		return nil
	}
	var ev *motor.Response
	if timeout <= 0 {
		ev = tp.WaitForNextEvent(time.Duration(timeout))
	} else {
		ev = tp.WaitForNextEvent(time.Duration(timeout) * time.Millisecond)
	}
	return serialize(ev)
}

//export OONIEngineTaskGetResult
func OONIEngineTaskGetResult(task C.OONITask) *C.char {
	tp := handler.getTaskHandle(task)
	if tp == nil {
		return nil
	}
	result := tp.Result()
	return serialize(result)
}

//export OONIEngineTaskIsDone
func OONIEngineTaskIsDone(task C.OONITask) (out C.uint8_t) {
	tp := handler.getTaskHandle(task)
	if tp == nil {
		return
	}
	if !tp.IsDone() {
		out++
	}
	return
}

//export OONIEngineInterruptTask
func OONIEngineInterruptTask(task C.OONITask) {
	tp := handler.getTaskHandle(task)
	if tp == nil {
		return
	}
	tp.Interrupt()
}

//export OONIEngineFreeTask
func OONIEngineFreeTask(task C.OONITask) {
	tp := handler.getTaskHandle(task)
	if tp != nil {
		tp.Interrupt()
	}
	handler.delete(Handle(task))
}

func main() {
	// do nothing
}
