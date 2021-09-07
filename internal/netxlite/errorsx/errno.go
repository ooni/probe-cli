// Code generated by go generate; DO NOT EDIT.
// Generated: 2021-09-07 16:43:08.462721 +0200 CEST m=+0.105415376

package errorsx

//go:generate go run ./internal/generrno/

import (
	"errors"
	"syscall"
)

// This enumeration lists the failures defined at
// https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md
const (
	//
	// System errors
	//
	FailureOperationCanceled           = "operation_canceled"
	FailureConnectionRefused           = "connection_refused"
	FailureConnectionReset             = "connection_reset"
	FailureHostUnreachable             = "host_unreachable"
	FailureTimedOut                    = "timed_out"
	FailureAddressFamilyNotSupported   = "address_family_not_supported"
	FailureAddressInUse                = "address_in_use"
	FailureAddressNotAvailable         = "address_not_available"
	FailureAlreadyConnected            = "already_connected"
	FailureBadAddress                  = "bad_address"
	FailureBadFileDescriptor           = "bad_file_descriptor"
	FailureConnectionAborted           = "connection_aborted"
	FailureConnectionAlreadyInProgress = "connection_already_in_progress"
	FailureDestinationAddressRequired  = "destination_address_required"
	FailureInterrupted                 = "interrupted"
	FailureInvalidArgument             = "invalid_argument"
	FailureMessageSize                 = "message_size"
	FailureNetworkDown                 = "network_down"
	FailureNetworkReset                = "network_reset"
	FailureNetworkUnreachable          = "network_unreachable"
	FailureNoBufferSpace               = "no_buffer_space"
	FailureNoProtocolOption            = "no_protocol_option"
	FailureNotASocket                  = "not_a_socket"
	FailureNotConnected                = "not_connected"
	FailureOperationWouldBlock         = "operation_would_block"
	FailurePermissionDenied            = "permission_denied"
	FailureProtocolNotSupported        = "protocol_not_supported"
	FailureWrongProtocolType           = "wrong_protocol_type"

	//
	// Library errors
	//
	FailureDNSBogonError           = "dns_bogon_error"
	FailureDNSNXDOMAINError        = "dns_nxdomain_error"
	FailureEOFError                = "eof_error"
	FailureGenericTimeoutError     = "generic_timeout_error"
	FailureQUICIncompatibleVersion = "quic_incompatible_version"
	FailureSSLFailedHandshake      = "ssl_failed_handshake"
	FailureSSLInvalidHostname      = "ssl_invalid_hostname"
	FailureSSLUnknownAuthority     = "ssl_unknown_authority"
	FailureSSLInvalidCertificate   = "ssl_invalid_certificate"
	FailureJSONParseError          = "json_parse_error"
)

// classifySyscallError converts a syscall error to the
// proper OONI error. Returns the OONI error string
// on success, an empty string otherwise.
func classifySyscallError(err error) string {
	// filter out system errors: necessary to detect all windows errors
	// https://github.com/ooni/probe/issues/1526 describes the problem
	// of mapping localized windows errors.
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return ""
	}
	switch errno {
	case ECANCELED:
		return FailureOperationCanceled
	case ECONNREFUSED:
		return FailureConnectionRefused
	case ECONNRESET:
		return FailureConnectionReset
	case EHOSTUNREACH:
		return FailureHostUnreachable
	case ETIMEDOUT:
		return FailureTimedOut
	case EAFNOSUPPORT:
		return FailureAddressFamilyNotSupported
	case EADDRINUSE:
		return FailureAddressInUse
	case EADDRNOTAVAIL:
		return FailureAddressNotAvailable
	case EISCONN:
		return FailureAlreadyConnected
	case EFAULT:
		return FailureBadAddress
	case EBADF:
		return FailureBadFileDescriptor
	case ECONNABORTED:
		return FailureConnectionAborted
	case EALREADY:
		return FailureConnectionAlreadyInProgress
	case EDESTADDRREQ:
		return FailureDestinationAddressRequired
	case EINTR:
		return FailureInterrupted
	case EINVAL:
		return FailureInvalidArgument
	case EMSGSIZE:
		return FailureMessageSize
	case ENETDOWN:
		return FailureNetworkDown
	case ENETRESET:
		return FailureNetworkReset
	case ENETUNREACH:
		return FailureNetworkUnreachable
	case ENOBUFS:
		return FailureNoBufferSpace
	case ENOPROTOOPT:
		return FailureNoProtocolOption
	case ENOTSOCK:
		return FailureNotASocket
	case ENOTCONN:
		return FailureNotConnected
	case EWOULDBLOCK:
		return FailureOperationWouldBlock
	case EACCES:
		return FailurePermissionDenied
	case EPROTONOSUPPORT:
		return FailureProtocolNotSupported
	case EPROTOTYPE:
		return FailureWrongProtocolType
	}
	return ""
}
