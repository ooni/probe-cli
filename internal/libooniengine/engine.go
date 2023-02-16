package main

//
// C API
//

//#include <stdlib.h>
//
//#include "ooniengine.h"
import "C"

import (
	"unsafe"

	"github.com/ooni/probe-cli/v3/internal/version"
)

//export OONIEngineVersion
func OONIEngineVersion() *C.char {
	return C.CString(version.Version)
}

//export OONIEngineFreeMemory
func OONIEngineFreeMemory(ptr *C.void) {
	C.free(unsafe.Pointer(ptr))
}

func main() {
	// do nothing
}
