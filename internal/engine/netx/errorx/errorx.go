// Package errorx contains error extensions
package errorx

import (
	"errors"
)

const (
	// FailureConnectionRefused means ECONNREFUSED.
	FailureConnectionRefused = "connection_refused"

	// FailureConnectionReset means ECONNRESET.
	FailureConnectionReset = "connection_reset"

	// FailureDNSBogonError means we detected bogon in DNS reply.
	FailureDNSBogonError = "dns_bogon_error"

	// FailureDNSNXDOMAINError means we got NXDOMAIN in DNS reply.
	FailureDNSNXDOMAINError = "dns_nxdomain_error"

	// FailureEOFError means we got unexpected EOF on connection.
	FailureEOFError = "eof_error"

	// FailureGenericTimeoutError means we got some timer has expired.
	FailureGenericTimeoutError = "generic_timeout_error"

	// FailureInterrupted means that the user interrupted us.
	FailureInterrupted = "interrupted"

	// FailureNoCompatibleQUICVersion means that the server does not support the proposed QUIC version
	FailureNoCompatibleQUICVersion = "quic_incompatible_version"

	// FailureSSLInvalidHostname means we got certificate is not valid for SNI.
	FailureSSLInvalidHostname = "ssl_invalid_hostname"

	// FailureSSLUnknownAuthority means we cannot find CA validating certificate.
	FailureSSLUnknownAuthority = "ssl_unknown_authority"

	// FailureSSLInvalidCertificate means certificate experired or other
	// sort of errors causing it to be invalid.
	FailureSSLInvalidCertificate = "ssl_invalid_certificate"

	// FailureJSONParseError indicates that we couldn't parse a JSON
	FailureJSONParseError = "json_parse_error"
)

const (
	// ResolveOperation is the operation where we resolve a domain name
	ResolveOperation = "resolve"

	// ConnectOperation is the operation where we do a TCP connect
	ConnectOperation = "connect"

	// TLSHandshakeOperation is the TLS handshake
	TLSHandshakeOperation = "tls_handshake"

	// QUICHandshakeOperation is the handshake to setup a QUIC connection
	QUICHandshakeOperation = "quic_handshake"

	// HTTPRoundTripOperation is the HTTP round trip
	HTTPRoundTripOperation = "http_round_trip"

	// CloseOperation is when we close a socket
	CloseOperation = "close"

	// ReadOperation is when we read from a socket
	ReadOperation = "read"

	// WriteOperation is when we write to a socket
	WriteOperation = "write"

	// ReadFromOperation is when we read from an UDP socket
	ReadFromOperation = "read_from"

	// WriteToOperation is when we write to an UDP socket
	WriteToOperation = "write_to"

	// UnknownOperation is when we cannot determine the operation
	UnknownOperation = "unknown"

	// TopLevelOperation is used when the failure happens at top level. This
	// happens for example with urlgetter with a cancelled context.
	TopLevelOperation = "top_level"
)

// ErrDNSBogon indicates that we found a bogon address. This is the
// correct value with which to initialize MeasurementRoot.ErrDNSBogon
// to tell this library to return an error when a bogon is found.
var ErrDNSBogon = errors.New("dns: detected bogon address")

// ErrWrapper is our error wrapper for Go errors. The key objective of
// this structure is to properly set Failure, which is also returned by
// the Error() method, so be one of the OONI defined strings.
type ErrWrapper struct {
	// ConnID is the connection ID, or zero if not known.
	ConnID int64

	// DialID is the dial ID, or zero if not known.
	DialID int64

	// Failure is the OONI failure string. The failure strings are
	// loosely backward compatible with Measurement Kit.
	//
	// This is either one of the FailureXXX strings or any other
	// string like `unknown_failure ...`. The latter represents an
	// error that we have not yet mapped to a failure.
	Failure string

	// Operation is the operation that failed. If possible, it
	// SHOULD be a _major_ operation. Major operations are:
	//
	// - ResolveOperation: resolving a domain name failed
	// - ConnectOperation: connecting to an IP failed
	// - TLSHandshakeOperation: TLS handshaking failed
	// - HTTPRoundTripOperation: other errors during round trip
	//
	// Because a network connection doesn't necessarily know
	// what is the current major operation we also have the
	// following _minor_ operations:
	//
	// - CloseOperation: CLOSE failed
	// - ReadOperation: READ failed
	// - WriteOperation: WRITE failed
	//
	// If an ErrWrapper referring to a major operation is wrapping
	// another ErrWrapper and such ErrWrapper already refers to
	// a major operation, then the new ErrWrapper should use the
	// child ErrWrapper major operation. Otherwise, it should use
	// its own major operation. This way, the topmost wrapper is
	// supposed to refer to the major operation that failed.
	Operation string

	// TransactionID is the transaction ID, or zero if not known.
	TransactionID int64

	// WrappedErr is the error that we're wrapping.
	WrappedErr error
}

// Error returns a description of the error that occurred.
func (e *ErrWrapper) Error() string {
	return e.Failure
}

// Unwrap allows to access the underlying error
func (e *ErrWrapper) Unwrap() error {
	return e.WrappedErr
}

// SafeErrWrapperBuilder contains a builder for ErrWrapper that
// is safe, i.e., behaves correctly when the error is nil.
type SafeErrWrapperBuilder struct {
	// ConnID is the connection ID, if any
	ConnID int64

	// DialID is the dial ID, if any
	DialID int64

	// Error is the error, if any
	Error error

	// Operation is the operation that failed
	Operation string

	// TransactionID is the transaction ID, if any
	TransactionID int64
}

// MaybeBuild builds a new ErrWrapper, if b.Error is not nil, and returns
// a nil error value, instead, if b.Error is nil.
func (b SafeErrWrapperBuilder) MaybeBuild() (err error) {
	if b.Error != nil {
		err = &ErrWrapper{
			ConnID:        b.ConnID,
			DialID:        b.DialID,
			TransactionID: b.TransactionID,
			WrappedErr:    b.Error,
		}
	}
	return
}
