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
	"runtime/cgo"
	"time"
	"unsafe"

	"github.com/ooni/probe-cli/v3/internal/version"
)

const (
	// invalidTaskHandle represents the invalid task handle.
	invalidTaskHandle = 0
)

// parse converts a JSON request string to the concrete Go type.
func parse(req *C.char) (*request, error) {
	var out *request
	err := json.Unmarshal([]byte(C.GoString(req)), out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// serialize serializes a OONI response to a JSON string accessible to C code.
func serialize(resp *response) *C.char {
	out, err := json.Marshal(resp)
	if err != nil {
		log.Printf("serializeMessage: cannot serialize message: %s", err.Error())
		return C.CString("")
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
	taskName, err := resolveTask(r)
	if err != nil {
		log.Printf("OONIEngineCall: %s", err.Error())
		return invalidTaskHandle
	}
	tp := startTask(taskName, r)
	if tp == nil {
		log.Printf("OONITaskStart: startTask return NULL")
		return invalidTaskHandle
	}
	return C.OONITask(cgo.NewHandle(tp))
}

//export OONIEngineWaitForNextEvent
func OONIEngineWaitForNextEvent(task C.OONITask, timeout C.int32_t) *C.char {
	tp := cgo.Handle(task).Value().(taskAPI)
	ev := tp.waitForNextEvent(time.Duration(timeout) * time.Millisecond)
	return serialize(ev)
}

//export OONIEngineInterrupt
func OONIEngineInterrupt(task C.OONITask) {
	tp := cgo.Handle(task).Value().(taskAPI)
	tp.interrupt()
}

func main() {
	// do nothing
}
