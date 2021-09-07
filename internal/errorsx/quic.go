package errorsx

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// QUICContextDialer is a dialer for QUIC using Context.
type QUICContextDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	DialContext(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)
}

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening UDPConn.
	Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error)
}

// ErrorWrapperQUICListener is a QUICListener that wraps errors.
type ErrorWrapperQUICListener struct {
	// QUICListener is the underlying listener.
	QUICListener QUICListener
}

var _ QUICListener = &ErrorWrapperQUICListener{}

// Listen implements QUICListener.Listen.
func (qls *ErrorWrapperQUICListener) Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, SafeErrWrapperBuilder{
			Error:     err,
			Operation: QUICListenOperation,
		}.MaybeBuild()
	}
	return &errorWrapperUDPConn{pconn}, nil
}

// errorWrapperUDPConn is a quicx.UDPLikeConn that wraps errors.
type errorWrapperUDPConn struct {
	// UDPLikeConn is the underlying conn.
	quicx.UDPLikeConn
}

var _ quicx.UDPLikeConn = &errorWrapperUDPConn{}

// WriteTo implements quicx.UDPLikeConn.WriteTo.
func (c *errorWrapperUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	if err != nil {
		return 0, SafeErrWrapperBuilder{
			Error:     err,
			Operation: WriteToOperation,
		}.MaybeBuild()
	}
	return count, nil
}

// ReadFrom implements quicx.UDPLikeConn.ReadFrom.
func (c *errorWrapperUDPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.UDPLikeConn.ReadFrom(b)
	if err != nil {
		return 0, nil, SafeErrWrapperBuilder{
			Error:     err,
			Operation: ReadFromOperation,
		}.MaybeBuild()
	}
	return n, addr, nil
}

// ErrorWrapperQUICDialer is a dialer that performs quic err wrapping
type ErrorWrapperQUICDialer struct {
	Dialer QUICContextDialer
}

// DialContext implements ContextDialer.DialContext
func (d *ErrorWrapperQUICDialer) DialContext(
	ctx context.Context, network string, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	sess, err := d.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	err = SafeErrWrapperBuilder{
		Classifier: classifyQUICFailure,
		Error:      err,
		Operation:  QUICHandshakeOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// classifyQUICFailure is a classifier to translate QUIC errors to OONI error strings.
func classifyQUICFailure(err error) string {
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
	return toFailureString(err)
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
