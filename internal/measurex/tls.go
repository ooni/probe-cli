package measurex

//
// TLS
//
// Wraps TLS code to write events into a WritableDB.
//

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TLSHandshaker performs TLS handshakes.
type TLSHandshaker = netxlite.TLSHandshaker

// WrapTLSHandshaker wraps a netxlite.TLSHandshaker to return a new
// instance of TLSHandshaker that saves events into the DB.
func (mx *Measurer) WrapTLSHandshaker(db WritableDB, thx netxlite.TLSHandshaker) TLSHandshaker {
	return &tlsHandshakerDB{TLSHandshaker: thx, db: db, begin: mx.Begin}
}

// NewTLSHandshakerStdlib creates a new TLS handshaker that
// saves results into the DB and uses the stdlib for TLS.
func (mx *Measurer) NewTLSHandshakerStdlib(db WritableDB, logger Logger) TLSHandshaker {
	return mx.WrapTLSHandshaker(db, netxlite.NewTLSHandshakerStdlib(logger))
}

type tlsHandshakerDB struct {
	netxlite.TLSHandshaker
	begin time.Time
	db    WritableDB
}

// TLSHandshakeEvent contains a TLS handshake event.
type TLSHandshakeEvent struct {
	// JSON names compatible with df-006-tlshandshake
	CipherSuite     string                `json:"cipher_suite"`
	Failure         *string               `json:"failure"`
	NegotiatedProto string                `json:"negotiated_proto"`
	TLSVersion      string                `json:"tls_version"`
	PeerCerts       []*ArchivalBinaryData `json:"peer_certificates"`
	Finished        float64               `json:"t"`

	// JSON names that are consistent with the
	// spirit of the spec but are not in it
	RemoteAddr string   `json:"address"`
	SNI        string   `json:"server_name"` // used in prod
	ALPN       []string `json:"alpn"`
	SkipVerify bool     `json:"no_tls_verify"` // used in prod
	Oddity     Oddity   `json:"oddity"`
	Network    string   `json:"proto"`
	Started    float64  `json:"started"`
}

func (thx *tlsHandshakerDB) Handshake(ctx context.Context,
	conn Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
	network := conn.RemoteAddr().Network()
	remoteAddr := conn.RemoteAddr().String()
	started := time.Since(thx.begin).Seconds()
	tconn, state, err := thx.TLSHandshaker.Handshake(ctx, conn, config)
	finished := time.Since(thx.begin).Seconds()
	thx.db.InsertIntoTLSHandshake(&TLSHandshakeEvent{
		Network:         network,
		RemoteAddr:      remoteAddr,
		SNI:             config.ServerName,
		ALPN:            config.NextProtos,
		SkipVerify:      config.InsecureSkipVerify,
		Started:         started,
		Finished:        finished,
		Failure:         NewArchivalFailure(err),
		Oddity:          thx.computeOddity(err),
		TLSVersion:      netxlite.TLSVersionString(state.Version),
		CipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		NegotiatedProto: state.NegotiatedProtocol,
		PeerCerts:       NewArchivalTLSCerts(peerCerts(err, &state)),
	})
	return tconn, state, err
}

func (thx *tlsHandshakerDB) computeOddity(err error) Oddity {
	if err == nil {
		return ""
	}
	switch err.Error() {
	case netxlite.FailureGenericTimeoutError:
		return OddityTLSHandshakeTimeout
	case netxlite.FailureConnectionReset:
		return OddityTLSHandshakeReset
	case netxlite.FailureEOFError:
		return OddityTLSHandshakeUnexpectedEOF
	case netxlite.FailureSSLInvalidHostname:
		return OddityTLSHandshakeInvalidHostname
	case netxlite.FailureSSLUnknownAuthority:
		return OddityTLSHandshakeUnknownAuthority
	default:
		return OddityTLSHandshakeOther
	}
}

func peerCerts(err error, state *tls.ConnectionState) (out [][]byte) {
	var x509HostnameError x509.HostnameError
	if errors.As(err, &x509HostnameError) {
		// Test case: https://wrong.host.badssl.com/
		return [][]byte{x509HostnameError.Certificate.Raw}
	}
	var x509UnknownAuthorityError x509.UnknownAuthorityError
	if errors.As(err, &x509UnknownAuthorityError) {
		// Test case: https://self-signed.badssl.com/. This error has
		// never been among the ones returned by MK.
		return [][]byte{x509UnknownAuthorityError.Cert.Raw}
	}
	var x509CertificateInvalidError x509.CertificateInvalidError
	if errors.As(err, &x509CertificateInvalidError) {
		// Test case: https://expired.badssl.com/
		return [][]byte{x509CertificateInvalidError.Cert.Raw}
	}
	for _, cert := range state.PeerCertificates {
		out = append(out, cert.Raw)
	}
	return
}
