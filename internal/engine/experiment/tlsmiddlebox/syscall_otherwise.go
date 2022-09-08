//go:build !cgo

package tlsmiddlebox

//
// Disabled CGO for SO_ERROR
//

// getErrFromSockOpt returns the errno of the SO_ERROR
//
// This is the CGO_ENABLED=0 implementation of this function, which
// always returns errno=0 for SO_ERROR
func getErrFromSockOpt(fd uintptr) int {
	return 0
}
