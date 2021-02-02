// Package errorx contains error extensions
package errorx

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"regexp"
	"strings"
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
			Failure:       toFailureString(b.Error),
			Operation:     toOperationString(b.Error, b.Operation),
			TransactionID: b.TransactionID,
			WrappedErr:    b.Error,
		}
	}
	return
}

func toFailureString(err error) string {
	// The list returned here matches the values used by MK unless
	// explicitly noted otherwise with a comment.

	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}

	if errors.Is(err, ErrDNSBogon) {
		return FailureDNSBogonError // not in MK
	}
	if errors.Is(err, context.Canceled) {
		return FailureInterrupted
	}
	var x509HostnameError x509.HostnameError
	if errors.As(err, &x509HostnameError) {
		// Test case: https://wrong.host.badssl.com/
		return FailureSSLInvalidHostname
	}
	var x509UnknownAuthorityError x509.UnknownAuthorityError
	if errors.As(err, &x509UnknownAuthorityError) {
		// Test case: https://self-signed.badssl.com/. This error has
		// never been among the ones returned by MK.
		return FailureSSLUnknownAuthority
	}
	var x509CertificateInvalidError x509.CertificateInvalidError
	if errors.As(err, &x509CertificateInvalidError) {
		// Test case: https://expired.badssl.com/
		return FailureSSLInvalidCertificate
	}

	s := err.Error()
	if strings.HasSuffix(s, "operation was canceled") {
		return FailureInterrupted
	}
	if strings.HasSuffix(s, "EOF") {
		return FailureEOFError
	}
	if strings.HasSuffix(s, "connection refused") {
		return FailureConnectionRefused
	}
	if strings.HasSuffix(s, "connection reset by peer") {
		return FailureConnectionReset
	}
	if strings.HasSuffix(s, "context deadline exceeded") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "transaction is timed out") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "i/o timeout") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "TLS handshake timeout") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "no such host") {
		// This is dns_lookup_error in MK but such error is used as a
		// generic "hey, the lookup failed" error. Instead, this error
		// that we return here is significantly more specific.
		return FailureDNSNXDOMAINError
	}

	// TODO(kelmenhorst): see whether it is possible to match errors
	// from qtls rather than strings for TLS errors below.
	//
	// TODO(kelmenhorst): make sure we have tests for all errors. Also,
	// how to ensure we are robust to changes in other libs?
	//
	// special QUIC errors
	matched, err := regexp.MatchString(`.*x509: certificate is valid for.*not.*`, s)
	if matched {
		return FailureSSLInvalidHostname
	}
	if strings.HasSuffix(s, "x509: certificate signed by unknown authority") {
		return FailureSSLUnknownAuthority
	}
	certInvalidErrors := []string{"x509: certificate is not authorized to sign other certificates", "x509: certificate has expired or is not yet valid:", "x509: a root or intermediate certificate is not authorized to sign for this name:", "x509: a root or intermediate certificate is not authorized for an extended key usage:", "x509: too many intermediates for path length constraint", "x509: certificate specifies an incompatible key usage", "x509: issuer name does not match subject from issuing certificate", "x509: issuer has name constraints but leaf doesn't have a SAN extension", "x509: issuer has name constraints but leaf contains unknown or unconstrained name:"}
	for _, errstr := range certInvalidErrors {
		if strings.Contains(s, errstr) {
			return FailureSSLInvalidCertificate
		}
	}
	if strings.HasPrefix(s, "No compatible QUIC version found") {
		return FailureNoCompatibleQUICVersion
	}
	if strings.HasSuffix(s, "Handshake did not complete in time") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "connection_refused") {
		return FailureConnectionRefused
	}
	if strings.Contains(s, "stateless_reset") {
		return FailureConnectionReset
	}
	if strings.Contains(s, "deadline exceeded") {
		return FailureGenericTimeoutError
	}
	formatted := fmt.Sprintf("unknown_failure: %s", s)
	return Scrub(formatted) // scrub IP addresses in the error
}

func toOperationString(err error, operation string) string {
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		// Basically, as explained in ErrWrapper docs, let's
		// keep the child major operation, if any.
		if errwrapper.Operation == ConnectOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == HTTPRoundTripOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == ResolveOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == TLSHandshakeOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == QUICHandshakeOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == "quic_handshake_start" {
			return QUICHandshakeOperation
		}
		if errwrapper.Operation == "quic_handshake_done" {
			return QUICHandshakeOperation
		}
		// FALLTHROUGH
	}
	return operation
}
