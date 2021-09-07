package errorsx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
)

// TLSHandshaker is the generic TLS handshaker
type TLSHandshaker interface {
	Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}

// ErrorWrapperTLSHandshaker wraps the returned error to be an OONI error
type ErrorWrapperTLSHandshaker struct {
	TLSHandshaker
}

// Handshake implements TLSHandshaker.Handshake
func (h *ErrorWrapperTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	err = SafeErrWrapperBuilder{
		Classifier: ClassifyTLSHandshakeError,
		Error:      err,
		Operation:  TLSHandshakeOperation,
	}.MaybeBuild()
	return tlsconn, state, err
}

// ClassifyTLSHandshakeError maps an error occurred during the TLS
// handshake to an OONI failure string.
func ClassifyTLSHandshakeError(err error) string {
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
