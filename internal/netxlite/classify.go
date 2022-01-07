package netxlite

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/scrubber"
)

// classifyGenericError is maps an error occurred during an operation
// to an OONI failure string. This specific classifier is the most
// generic one. You usually use it when mapping I/O errors. You should
// check whether there is a specific classifier for more specific
// operations (e.g., DNS resolution, TLS handshake).
//
// If the input error is an *ErrWrapper we don't perform
// the classification again and we return its Failure.
//
// We put inside this classifier:
//
// - system call errors;
//
// - generic errors that can occur in multiple places;
//
// - all the errors that depend on strings.
//
// The more specific classifiers will call this classifier if
// they fail to find a mapping for the input error.
//
// If everything else fails, this classifier returns a string
// like "unknown_failure: XXX" where XXX has been scrubbed
// so to remove any network endpoints from the original error string.
func classifyGenericError(err error) string {
	// The list returned here matches the values used by MK unless
	// explicitly noted otherwise with a comment.

	// QUIRK: we cannot remove this check as long as this function
	// is exported and used independently from NewErrWrapper.
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}

	// Classify system errors first. We could use strings for many
	// of them on Unix, but this would fail on Windows as described
	// by https://github.com/ooni/probe/issues/1526.
	if failure := classifySyscallError(err); failure != "" {
		return failure
	}

	if errors.Is(err, context.Canceled) {
		return FailureInterrupted
	}

	if failure := classifyWithStringSuffix(err); failure != "" {
		return failure
	}

	formatted := fmt.Sprintf("unknown_failure: %s", err.Error())
	return scrubber.Scrub(formatted) // scrub IP addresses in the error
}

// classifyWithStringSuffix is a subset of ClassifyGenericError that
// performs classification by looking at error suffixes. This function
// will return an empty string if it cannot classify the error.
func classifyWithStringSuffix(err error) string {
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
	if strings.HasSuffix(s, "TLS handshake timeout") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, DNSNoSuchHostSuffix) {
		// This is dns_lookup_error in MK but such error is used as a
		// generic "hey, the lookup failed" error. Instead, this error
		// that we return here is significantly more specific.
		return FailureDNSNXDOMAINError
	}
	if strings.HasSuffix(s, DNSServerMisbehavingSuffix) {
		return FailureDNSServerMisbehaving
	}
	if strings.HasSuffix(s, DNSNoAnswerSuffix) {
		return FailureDNSNoAnswer
	}
	if strings.HasSuffix(s, "use of closed network connection") {
		return FailureConnectionAlreadyClosed
	}
	return "" // not found
}

// TLS alert protocol as defined in RFC8446. We need these definitions
// to figure out which error occurred during a QUIC handshake.
const (
	// Sender was unable to negotiate an acceptable set of security parameters given the options available.
	quicTLSAlertHandshakeFailure = 40

	// Certificate was corrupt, contained signatures that did not verify correctly, etc.
	quicTLSAlertBadCertificate = 42

	// Certificate was of an unsupported type.
	quicTLSAlertUnsupportedCertificate = 43

	// Certificate was revoked by its signer.
	quicTLSAlertCertificateRevoked = 44

	// Certificate has expired or is not currently valid.
	quicTLSAlertCertificateExpired = 45

	// Some unspecified issue arose in processing the certificate, rendering it unacceptable.
	quicTLSAlertCertificateUnknown = 46

	// Certificate was not accepted because the CA certificate could not be located or could not be matched with a known trust anchor.
	quicTLSAlertUnknownCA = 48

	// Handshake (not record layer) cryptographic operation failed.
	quicTLSAlertDecryptError = 51

	// Sent by servers when no server exists identified by the name provided by the client via the "server_name" extension.
	quicTLSUnrecognizedName = 112
)

// classifyQUICHandshakeError maps errors during a QUIC
// handshake to OONI failure strings.
//
// If the input error is an *ErrWrapper we don't perform
// the classification again and we return its Failure.
//
// If this classifier fails, it calls ClassifyGenericError
// and returns to the caller its return value.
func classifyQUICHandshakeError(err error) string {

	// QUIRK: we cannot remove this check as long as this function
	// is exported and used independently from NewErrWrapper.
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}

	var (
		versionNegotiation *quic.VersionNegotiationError
		statelessReset     *quic.StatelessResetError
		handshakeTimeout   *quic.HandshakeTimeoutError
		idleTimeout        *quic.IdleTimeoutError
		transportError     *quic.TransportError
	)

	if errors.As(err, &versionNegotiation) {
		return FailureQUICIncompatibleVersion
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
		if quicIsCertificateError(errCode) {
			return FailureSSLInvalidCertificate
		}
		// TLSAlertDecryptError and TLSAlertHandshakeFailure are summarized to a
		// FailureSSLHandshake error because both alerts are caused by a failed or
		// corrupted parameter negotiation during the TLS handshake.
		if errCode == quicTLSAlertDecryptError || errCode == quicTLSAlertHandshakeFailure {
			return FailureSSLFailedHandshake
		}
		if errCode == quicTLSAlertUnknownCA {
			return FailureSSLUnknownAuthority
		}
		if errCode == quicTLSUnrecognizedName {
			return FailureSSLInvalidHostname
		}

		// quic.TransportError wraps OONI errors using the error
		// code quic.InternalError. So, if the error code is
		// an internal error, search for a OONI error and, if
		// found, just return such an error.
		if transportError.ErrorCode == quic.InternalError {
			if s := failuresMap[transportError.ErrorMessage]; s != "" {
				return s
			}
		}
	}
	return classifyGenericError(err)
}

// quicIsCertificateError tells us whether a specific TLS alert error
// we received is actually an error depending on the certificate.
//
// The set of checks we implement here is a set of heuristics based
// on our understanding of the TLS spec and may need tweaks.
func quicIsCertificateError(alert uint8) bool {
	// List out each case separately so we know we test them
	switch alert {
	case quicTLSAlertBadCertificate:
		return true
	case quicTLSAlertUnsupportedCertificate:
		return true
	case quicTLSAlertCertificateExpired:
		return true
	case quicTLSAlertCertificateRevoked:
		return true
	case quicTLSAlertCertificateUnknown:
		return true
	default:
		return false
	}
}

// ErrDNSBogon indicates that we found a bogon address. Code that
// filters for DNS bogons MUST use this error.
var ErrDNSBogon = errors.New("dns: detected bogon address")

// We use these strings to string-match errors in the standard library
// and map such errors to OONI failures.
const (
	DNSNoSuchHostSuffix        = "no such host"
	DNSServerMisbehavingSuffix = "server misbehaving"
	DNSNoAnswerSuffix          = "no answer from DNS server"
)

// These errors are returned by custom DNSTransport instances (e.g.,
// DNSOverHTTPS and DNSOverUDP). Their suffix matches the equivalent
// unexported errors used by the Go standard library.
var (
	ErrOODNSNoSuchHost  = fmt.Errorf("ooniresolver: %s", DNSNoSuchHostSuffix)
	ErrOODNSRefused     = errors.New("ooniresolver: refused")
	ErrOODNSMisbehaving = fmt.Errorf("ooniresolver: %s", DNSServerMisbehavingSuffix)
	ErrOODNSNoAnswer    = fmt.Errorf("ooniresolver: %s", DNSNoAnswerSuffix)
)

// classifyResolverError maps DNS resolution errors to
// OONI failure strings.
//
// If the input error is an *ErrWrapper we don't perform
// the classification again and we return its Failure.
//
// If this classifier fails, it calls ClassifyGenericError and
// returns to the caller its return value.
func classifyResolverError(err error) string {

	// QUIRK: we cannot remove this check as long as this function
	// is exported and used independently from NewErrWrapper.
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}

	if errors.Is(err, ErrDNSBogon) {
		return FailureDNSBogonError // not in MK
	}
	// Implementation note: we match errors that share the same
	// string of the stdlib in the generic classifier.
	if errors.Is(err, ErrOODNSRefused) {
		return FailureDNSRefusedError // not in MK
	}
	return classifyGenericError(err)
}

// classifyTLSHandshakeError maps an error occurred during the TLS
// handshake to an OONI failure string.
//
// If the input error is an *ErrWrapper we don't perform
// the classification again and we return its Failure.
//
// If this classifier fails, it calls ClassifyGenericError and
// returns to the caller its return value.
func classifyTLSHandshakeError(err error) string {

	// QUIRK: we cannot remove this check as long as this function
	// is exported and used independently from NewErrWrapper.
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
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
	return classifyGenericError(err)
}
