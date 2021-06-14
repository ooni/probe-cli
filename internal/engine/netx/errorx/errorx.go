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

func IsOONIErr(s string) bool {
	arr := []string{
		FailureConnectionRefused,
		FailureConnectionReset,
		FailureDNSBogonError,
		FailureDNSNXDOMAINError,
		FailureEOFError,
		FailureGenericTimeoutError,
		FailureInterrupted,
		FailureNoCompatibleQUICVersion,
		FailureSSLInvalidHostname,
		FailureSSLUnknownAuthority,
		FailureSSLInvalidCertificate,
		FailureJSONParseError,
	}
	for _, str := range arr {
		if s == str {
			return true
		}
	}
	return false
}

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
