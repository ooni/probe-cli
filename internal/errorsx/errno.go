// Code generated by go generate; DO NOT EDIT.
// Generated: 2021-07-02 15:15:17.997258 +0200 CEST m=+0.110031584

package errorsx

//go:generate go run ./generator/

import (
	"errors"
	"syscall"
)

// toSyscallErr converts a syscall error to the
// proper OONI error. Returns the OONI error string
// on success, an empty string otherwise.
func toSyscallErr(err error) string {
	// filter out system errors: necessary to detect all windows errors
	// https://github.com/ooni/probe/issues/1526 describes the problem
	// of mapping localized windows errors.
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return ""
	}
	switch errno {
	case ECANCELED:
		return FailureInterrupted
	case ECONNREFUSED:
		return FailureConnectionRefused
	case ECONNRESET:
		return FailureConnectionReset
	case EHOSTUNREACH:
		return FailureHostUnreachable
	case ETIMEDOUT:
		return FailureGenericTimeoutError
	}
	return ""
}
