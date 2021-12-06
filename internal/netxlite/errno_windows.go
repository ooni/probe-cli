// Code generated by go generate; DO NOT EDIT.
// Generated: 2021-12-06 16:54:30.896609 +0100 CET m=+0.586686376

package netxlite

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// This enumeration provides a canonical name for
// every system-call error we support. Note: this list
// is system dependent. You're currently looking at
// the list of errors for windows.
const (
	ECONNREFUSED    = windows.WSAECONNREFUSED
	ECONNRESET      = windows.WSAECONNRESET
	EHOSTUNREACH    = windows.WSAEHOSTUNREACH
	ETIMEDOUT       = windows.WSAETIMEDOUT
	EAFNOSUPPORT    = windows.WSAEAFNOSUPPORT
	EADDRINUSE      = windows.WSAEADDRINUSE
	EADDRNOTAVAIL   = windows.WSAEADDRNOTAVAIL
	EISCONN         = windows.WSAEISCONN
	EFAULT          = windows.WSAEFAULT
	EBADF           = windows.WSAEBADF
	ECONNABORTED    = windows.WSAECONNABORTED
	EALREADY        = windows.WSAEALREADY
	EDESTADDRREQ    = windows.WSAEDESTADDRREQ
	EINTR           = windows.WSAEINTR
	EINVAL          = windows.WSAEINVAL
	EMSGSIZE        = windows.WSAEMSGSIZE
	ENETDOWN        = windows.WSAENETDOWN
	ENETRESET       = windows.WSAENETRESET
	ENETUNREACH     = windows.WSAENETUNREACH
	ENOBUFS         = windows.WSAENOBUFS
	ENOPROTOOPT     = windows.WSAENOPROTOOPT
	ENOTSOCK        = windows.WSAENOTSOCK
	ENOTCONN        = windows.WSAENOTCONN
	EWOULDBLOCK     = windows.WSAEWOULDBLOCK
	EACCES          = windows.WSAEACCES
	EPROTONOSUPPORT = windows.WSAEPROTONOSUPPORT
	EPROTOTYPE      = windows.WSAEPROTOTYPE
	WSANO_DATA      = windows.WSANO_DATA
	WSANO_RECOVERY  = windows.WSANO_RECOVERY
	WSATRY_AGAIN    = windows.WSATRY_AGAIN
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
	case windows.WSAECONNREFUSED:
		return FailureConnectionRefused
	case windows.WSAECONNRESET:
		return FailureConnectionReset
	case windows.WSAEHOSTUNREACH:
		return FailureHostUnreachable
	case windows.WSAETIMEDOUT:
		return FailureTimedOut
	case windows.WSAEAFNOSUPPORT:
		return FailureAddressFamilyNotSupported
	case windows.WSAEADDRINUSE:
		return FailureAddressInUse
	case windows.WSAEADDRNOTAVAIL:
		return FailureAddressNotAvailable
	case windows.WSAEISCONN:
		return FailureAlreadyConnected
	case windows.WSAEFAULT:
		return FailureBadAddress
	case windows.WSAEBADF:
		return FailureBadFileDescriptor
	case windows.WSAECONNABORTED:
		return FailureConnectionAborted
	case windows.WSAEALREADY:
		return FailureConnectionAlreadyInProgress
	case windows.WSAEDESTADDRREQ:
		return FailureDestinationAddressRequired
	case windows.WSAEINTR:
		return FailureInterrupted
	case windows.WSAEINVAL:
		return FailureInvalidArgument
	case windows.WSAEMSGSIZE:
		return FailureMessageSize
	case windows.WSAENETDOWN:
		return FailureNetworkDown
	case windows.WSAENETRESET:
		return FailureNetworkReset
	case windows.WSAENETUNREACH:
		return FailureNetworkUnreachable
	case windows.WSAENOBUFS:
		return FailureNoBufferSpace
	case windows.WSAENOPROTOOPT:
		return FailureNoProtocolOption
	case windows.WSAENOTSOCK:
		return FailureNotASocket
	case windows.WSAENOTCONN:
		return FailureNotConnected
	case windows.WSAEWOULDBLOCK:
		return FailureOperationWouldBlock
	case windows.WSAEACCES:
		return FailurePermissionDenied
	case windows.WSAEPROTONOSUPPORT:
		return FailureProtocolNotSupported
	case windows.WSAEPROTOTYPE:
		return FailureWrongProtocolType
	case windows.WSANO_DATA:
		return FailureDNSNoAnswer
	case windows.WSANO_RECOVERY:
		return FailureDNSNonRecoverableFailure
	case windows.WSATRY_AGAIN:
		return FailureDNSTemporaryFailure
	}
	return ""
}
