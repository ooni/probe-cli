// Package errorx contains error extensions
package errorx

// TODO: eventually we want to re-structure the error classification code by clearly separating the layers where the error occur:
//			- errno.go and errno_test.go: contain only the errno classifier (for system errors)
//          - qtls.go and qtls_test.go: contain qtls dialers, handshaker, classifier
//          - tls.go and tls_test.go: contain tls dialers, handshaker, classifier
//          - resolver.go and resolver_test.go: contain dialers and classifier for resolving

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"syscall"

	"github.com/lucas-clemente/quic-go"
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

	// FailureHostUnreachable means that there is "no route to host".
	FailureHostUnreachable = "host_unreachable"

	// FailureInterrupted means that the user interrupted us.
	FailureInterrupted = "interrupted"

	// FailureNoCompatibleQUICVersion means that the server does not support the proposed QUIC version
	FailureNoCompatibleQUICVersion = "quic_incompatible_version"

	// FailureSSLHandshake means that the negotiation of cryptographic parameters failed
	FailureSSLHandshake = "ssl_failed_handshake"

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

// TLS alert protocol as defined in RFC8446
const (
	// Sender was unable to negotiate an acceptable set of security parameters given the options available.
	TLSAlertHandshakeFailure = 40

	// Certificate was corrupt, contained signatures that did not verify correctly, etc.
	TLSAlertBadCertificate = 42

	// Certificate was of an unsupported type.
	TLSAlertUnsupportedCertificate = 43

	// Certificate was revoked by its signer.
	TLSAlertCertificateRevoked = 44

	// Certificate has expired or is not currently valid.
	TLSAlertCertificateExpired = 45

	// Some unspecified issue arose in processing the certificate, rendering it unacceptable.
	TLSAlertCertificateUnknown = 46

	// Certificate was not accepted because the CA certificate could not be located or could not be matched with a known trust anchor.
	TLSAlertUnknownCA = 48

	// Handshake (not record layer) cryptographic operation failed.
	TLSAlertDecryptError = 51

	// Sent by servers when no server exists identified by the name provided by the client via the "server_name" extension.
	TLSUnrecognizedName = 112
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

	// Classifier is the local error to string classifier. When there is no
	// configured classifier we will use the generic classifier.
	Classifier func(err error) string

	// Operation is the operation that failed
	Operation string

	// TransactionID is the transaction ID, if any
	TransactionID int64
}

// MaybeBuild builds a new ErrWrapper, if b.Error is not nil, and returns
// a nil error value, instead, if b.Error is nil.
func (b SafeErrWrapperBuilder) MaybeBuild() (err error) {
	if b.Error != nil {
		classifier := b.Classifier
		if classifier == nil {
			classifier = toFailureString
		}
		err = &ErrWrapper{
			ConnID:        b.ConnID,
			DialID:        b.DialID,
			Failure:       classifier(b.Error),
			Operation:     toOperationString(b.Error, b.Operation),
			TransactionID: b.TransactionID,
			WrappedErr:    b.Error,
		}
	}
	return
}

// TODO (kelmenhorst, bassosimone):
// Use errors.Is / errors.As more often, when possible, in this classifier.
// These methods are more robust to library changes than strings.
// errors.Is / errors.As can only be used when the error is exported.
func toFailureString(err error) string {
	// The list returned here matches the values used by MK unless
	// explicitly noted otherwise with a comment.

	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}

	// filter out system errors: necessary to detect all windows errors
	// https://github.com/ooni/probe/issues/1526 describes the problem of mapping localized windows errors
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case ECANCELED:
			return FailureInterrupted
		case ECONNRESET:
			return FailureConnectionReset
		case ECONNREFUSED:
			return FailureConnectionRefused
		case EHOSTUNREACH:
			return FailureHostUnreachable
		case ETIMEDOUT:
			return FailureGenericTimeoutError
			// TODO(kelmenhorst): find out if we need more system errors here
		}
	}
	if errors.Is(err, context.Canceled) {
		return FailureInterrupted
	}
	s := err.Error()
	if strings.HasSuffix(s, "operation was canceled") {
		return FailureInterrupted
	}
	if strings.HasSuffix(s, "EOF") {
		return FailureEOFError
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
	// TODO(kelmenhorst,bassosimone): this can probably be (re-)moved
	if strings.HasSuffix(s, "TLS handshake timeout") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "no such host") {
		// This is dns_lookup_error in MK but such error is used as a
		// generic "hey, the lookup failed" error. Instead, this error
		// that we return here is significantly more specific.
		return FailureDNSNXDOMAINError
	}
	formatted := fmt.Sprintf("unknown_failure: %s", s)
	return Scrub(formatted) // scrub IP addresses in the error
}

// ClassifyQUICFailure is a classifier to translate QUIC errors to OONI error strings.
// TODO(kelmenhorst,bassosimone): Consider moving this into quicdialer.
func ClassifyQUICFailure(err error) string {
	var versionNegotiation *quic.VersionNegotiationError
	var statelessReset *quic.StatelessResetError
	var handshakeTimeout *quic.HandshakeTimeoutError
	var idleTimeout *quic.IdleTimeoutError
	var transportError *quic.TransportError

	if errors.As(err, &versionNegotiation) {
		return FailureNoCompatibleQUICVersion
	}
	if errors.As(err, &statelessReset) {
		return FailureConnectionReset
	}
	if errors.As(err, &handshakeTimeout) {
		return FailureGenericTimeoutError
	}
	if errors.As(err, &idleTimeout) {
		return FailureGenericTimeoutError
	}
	if errors.As(err, &transportError) {
		if transportError.ErrorCode == quic.ConnectionRefused {
			return FailureConnectionRefused
		}
		// the TLS Alert constants are taken from RFC8446
		errCode := uint8(transportError.ErrorCode)
		if isCertificateError(errCode) {
			return FailureSSLInvalidCertificate
		}
		if errCode == TLSAlertDecryptError || errCode == TLSAlertHandshakeFailure {
			return FailureSSLHandshake
		}
		if errCode == TLSAlertUnknownCA {
			return FailureSSLUnknownAuthority
		}
		if errCode == TLSUnrecognizedName {
			return FailureSSLInvalidHostname
		}
	}
	return toFailureString(err)
}

// ClassifyResolveFailure is a classifier to translate DNS resolving errors to OONI error strings.
// TODO(kelmenhorst,bassosimone): Consider moving this into resolve.
func ClassifyResolveFailure(err error) string {
	if errors.Is(err, ErrDNSBogon) {
		return FailureDNSBogonError // not in MK
	}
	return toFailureString(err)
}

// ClassifyTLSFailure is a classifier to translate TLS errors to OONI error strings.
// TODO(kelmenhorst,bassosimone): Consider moving this into tlsdialer.
func ClassifyTLSFailure(err error) string {
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
	return toFailureString(err)
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

func isCertificateError(alert uint8) bool {
	return (alert == TLSAlertBadCertificate ||
		alert == TLSAlertUnsupportedCertificate ||
		alert == TLSAlertCertificateExpired ||
		alert == TLSAlertCertificateRevoked ||
		alert == TLSAlertCertificateUnknown)
}
