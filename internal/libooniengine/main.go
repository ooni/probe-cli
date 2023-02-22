package main

//
// C API
//

//#include <stdlib.h>
//
//#include "engine.h"
import "C"

import (
	"runtime/cgo"
	"unsafe"

	"github.com/ooni/probe-cli/v3/internal/version"
)

const (
	// invalidTaskHandle represents the invalid task handle.
	invalidTaskHandle = 0
)

//export OONIEngineVersion
func OONIEngineVersion() *C.char {
	return C.CString(version.Version)
}

//export OONIEngineFreeMemory
func OONIEngineFreeMemory(ptr *C.void) {
	C.free(unsafe.Pointer(ptr))
}

//export NewSession
func NewSession(config *C.char) C.OONITask {
	value := []byte(C.GoString(config))
	tp := startTask("NewSession", value)
	if tp == nil {
		return invalidTaskHandle
	}
	return C.OONITask(cgo.NewHandle(tp))
}

func main() {
	// do nothing
}
