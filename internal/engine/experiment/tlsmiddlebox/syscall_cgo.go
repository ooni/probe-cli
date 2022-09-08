//go:build cgo && windows

package tlsmiddlebox

//
// CGO support for SO_ERROR
//

/*
#cgo windows LDFLAGS: -lws2_32

#ifdef _WIN32
#include <winsock2.h>
#endif
*/
import "C"

import "unsafe"

// getErrFromSockOpt returns the errno of the SO_ERROR
//
// This is the CGO_ENABLED=1 implementation of this function, which
// returns the errno obtained from the getsockopt call
func getErrFromSockOpt(fd uintptr) int {
	var cErrno C.int
	szInt := C.sizeof_int
	C.getsockopt((C.SOCKET)(fd), (C.int)(C.SOL_SOCKET), (C.int)(C.SO_ERROR), (*C.char)(unsafe.Pointer(&cErrno)), (*C.int)(unsafe.Pointer(&szInt)))
	return int(cErrno)
}
