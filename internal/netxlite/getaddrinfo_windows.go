//go:build cgo && windows

package netxlite

//#include <ws2tcpip.h>
import "C"

import "syscall"

const getaddrinfoAIFlags = C.AI_CANONNAME

// Making constants available to Go code so we can run tests (it seems
// it's not possible to import C directly in tests, sadly).
const (
	aiCanonname = C.AI_CANONNAME
)

// toError is the function that converts the return value from
// the getaddrinfo function into a proper Go error.
func (state *getaddrinfoState) toError(code int64, err error, goos string) error {
	if err == nil {
		// Implementation note: on Windows getaddrinfo directly
		// returns what is basically a winsock2 error. So if there
		// is no other error, just cast code to a syscall err.
		err = syscall.Errno(code)
	}
	return newErrGetaddrinfo(int64(code), err)
}
