package errorsx

// This enumeration lists the non-syscall-derived failures defined at
// https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md
//
// See also the enumeration at errno.go for the failures that
// derive directly from system call errors.
const (
	// FailureDNSBogonError means we detected bogon in DNS reply.
	FailureDNSBogonError = "dns_bogon_error"

	// FailureDNSNXDOMAINError means we got NXDOMAIN in DNS reply.
	FailureDNSNXDOMAINError = "dns_nxdomain_error"

	// FailureEOFError means we got unexpected EOF on connection.
	FailureEOFError = "eof_error"

	// FailureGenericTimeoutError means we got some timer has expired.
	FailureGenericTimeoutError = "generic_timeout_error"

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
