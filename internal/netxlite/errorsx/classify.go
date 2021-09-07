package errorsx

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/scrubber"
)

// TODO (kelmenhorst, bassosimone):
// Use errors.Is / errors.As more often, when possible, in this classifier.
// These methods are more robust to library changes than strings.
// errors.Is / errors.As can only be used when the error is exported.

// ClassifyGenericError is the generic classifier mapping an error
// occurred during an operation to an OONI failure string.
func ClassifyGenericError(err error) string {
	// The list returned here matches the values used by MK unless
	// explicitly noted otherwise with a comment.

	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}

	if failure := classifySyscallError(err); failure != "" {
		return failure
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
	// TODO(kelmenhorst,bassosimone): this can probably be moved since it's TLS specific
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
	return scrubber.Scrub(formatted) // scrub IP addresses in the error
}

// TLS alert protocol as defined in RFC8446
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

func quicIsCertificateError(alert uint8) bool {
	return (alert == quicTLSAlertBadCertificate ||
		alert == quicTLSAlertUnsupportedCertificate ||
		alert == quicTLSAlertCertificateExpired ||
		alert == quicTLSAlertCertificateRevoked ||
		alert == quicTLSAlertCertificateUnknown)
}

// ClassifyQUICHandshakeError maps an error occurred during the QUIC
// handshake to an OONI failure string.
func ClassifyQUICHandshakeError(err error) string {
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}

	var versionNegotiation *quic.VersionNegotiationError
	var statelessReset *quic.StatelessResetError
	var handshakeTimeout *quic.HandshakeTimeoutError
	var idleTimeout *quic.IdleTimeoutError
	var transportError *quic.TransportError

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
		// TLSAlertDecryptError and TLSAlertHandshakeFailure are summarized to a FailureSSLHandshake error because both
		// alerts are caused by a failed or corrupted parameter negotiation during the TLS handshake.
		if errCode == quicTLSAlertDecryptError || errCode == quicTLSAlertHandshakeFailure {
			return FailureSSLFailedHandshake
		}
		if errCode == quicTLSAlertUnknownCA {
			return FailureSSLUnknownAuthority
		}
		if errCode == quicTLSUnrecognizedName {
			return FailureSSLInvalidHostname
		}
	}
	return ClassifyGenericError(err)
}

// ErrDNSBogon indicates that we found a bogon address. This is the
// correct value with which to initialize MeasurementRoot.ErrDNSBogon
// to tell this library to return an error when a bogon is found.
var ErrDNSBogon = errors.New("dns: detected bogon address")

// ClassifyResolverError maps an error occurred during a domain name
// resolution to the corresponding OONI failure string.
func ClassifyResolverError(err error) string {
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}
	if errors.Is(err, ErrDNSBogon) {
		return FailureDNSBogonError // not in MK
	}
	return ClassifyGenericError(err)
}

// ClassifyTLSHandshakeError maps an error occurred during the TLS
// handshake to an OONI failure string.
func ClassifyTLSHandshakeError(err error) string {
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
	return ClassifyGenericError(err)
}
