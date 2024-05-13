// Code generated by go generate; DO NOT EDIT.
// Generated: 2024-05-13 18:43:23.377383 +0200 CEST m=+0.343667459

package netxlite

import (
	"errors"
	"syscall"

	"golang.org/x/sys/unix"
)

// This enumeration provides a canonical name for
// every system-call error we support. Note: this list
// is system dependent. You're currently looking at
// the list of errors for openbsd.
const (
	ECONNREFUSED    = unix.ECONNREFUSED
	ECONNRESET      = unix.ECONNRESET
	EHOSTUNREACH    = unix.EHOSTUNREACH
	ETIMEDOUT       = unix.ETIMEDOUT
	EAFNOSUPPORT    = unix.EAFNOSUPPORT
	EADDRINUSE      = unix.EADDRINUSE
	EADDRNOTAVAIL   = unix.EADDRNOTAVAIL
	EISCONN         = unix.EISCONN
	EFAULT          = unix.EFAULT
	EBADF           = unix.EBADF
	ECONNABORTED    = unix.ECONNABORTED
	EALREADY        = unix.EALREADY
	EDESTADDRREQ    = unix.EDESTADDRREQ
	EINTR           = unix.EINTR
	EINVAL          = unix.EINVAL
	EMSGSIZE        = unix.EMSGSIZE
	ENETDOWN        = unix.ENETDOWN
	ENETRESET       = unix.ENETRESET
	ENETUNREACH     = unix.ENETUNREACH
	ENOBUFS         = unix.ENOBUFS
	ENOPROTOOPT     = unix.ENOPROTOOPT
	ENOTSOCK        = unix.ENOTSOCK
	ENOTCONN        = unix.ENOTCONN
	EWOULDBLOCK     = unix.EWOULDBLOCK
	EACCES          = unix.EACCES
	EPROTONOSUPPORT = unix.EPROTONOSUPPORT
	EPROTOTYPE      = unix.EPROTOTYPE
)

// classifySyscallError converts a syscall error to the
// proper OONI error. Returns the OONI error string
// on success, an empty string otherwise.
func classifySyscallError(err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return ""
	}
	switch errno {
	case unix.ECONNREFUSED:
		return FailureConnectionRefused
	case unix.ECONNRESET:
		return FailureConnectionReset
	case unix.EHOSTUNREACH:
		return FailureHostUnreachable
	case unix.ETIMEDOUT:
		return FailureTimedOut
	case unix.EAFNOSUPPORT:
		return FailureAddressFamilyNotSupported
	case unix.EADDRINUSE:
		return FailureAddressInUse
	case unix.EADDRNOTAVAIL:
		return FailureAddressNotAvailable
	case unix.EISCONN:
		return FailureAlreadyConnected
	case unix.EFAULT:
		return FailureBadAddress
	case unix.EBADF:
		return FailureBadFileDescriptor
	case unix.ECONNABORTED:
		return FailureConnectionAborted
	case unix.EALREADY:
		return FailureConnectionAlreadyInProgress
	case unix.EDESTADDRREQ:
		return FailureDestinationAddressRequired
	case unix.EINTR:
		return FailureInterrupted
	case unix.EINVAL:
		return FailureInvalidArgument
	case unix.EMSGSIZE:
		return FailureMessageSize
	case unix.ENETDOWN:
		return FailureNetworkDown
	case unix.ENETRESET:
		return FailureNetworkReset
	case unix.ENETUNREACH:
		return FailureNetworkUnreachable
	case unix.ENOBUFS:
		return FailureNoBufferSpace
	case unix.ENOPROTOOPT:
		return FailureNoProtocolOption
	case unix.ENOTSOCK:
		return FailureNotASocket
	case unix.ENOTCONN:
		return FailureNotConnected
	case unix.EWOULDBLOCK:
		return FailureOperationWouldBlock
	case unix.EACCES:
		return FailurePermissionDenied
	case unix.EPROTONOSUPPORT:
		return FailureProtocolNotSupported
	case unix.EPROTOTYPE:
		return FailureWrongProtocolType
	}
	return ""
}
