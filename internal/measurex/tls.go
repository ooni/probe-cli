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

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// WrapTLSHandshaker wraps a netxlite.TLSHandshaker to return a new
// instance of TLSHandshaker that saves events into the DB.
func (mx *Measurer) WrapTLSHandshaker(db WritableDB, thx model.TLSHandshaker) model.TLSHandshaker {
	return &tlsHandshakerDB{TLSHandshaker: thx, db: db, begin: mx.Begin}
}

// NewTLSHandshakerStdlib creates a new TLS handshaker that
// saves results into the DB and uses the stdlib for TLS.
func (mx *Measurer) NewTLSHandshakerStdlib(db WritableDB, logger model.Logger) model.TLSHandshaker {
	return mx.WrapTLSHandshaker(db, netxlite.NewTLSHandshakerStdlib(logger))
}

type tlsHandshakerDB struct {
	model.TLSHandshaker
	begin time.Time
	db    WritableDB
}

// QUICTLSHandshakeEvent contains a QUIC or TLS handshake event.
type QUICTLSHandshakeEvent struct {
	CipherSuite     string
	Failure         *string
	NegotiatedProto string
	TLSVersion      string
	PeerCerts       [][]byte
	Finished        float64
	RemoteAddr      string
	SNI             string
	ALPN            []string
	SkipVerify      bool
	Network         string
	Started         float64
}

func (thx *tlsHandshakerDB) Handshake(ctx context.Context,
	conn Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
	network := conn.RemoteAddr().Network()
	remoteAddr := conn.RemoteAddr().String()
	started := time.Since(thx.begin).Seconds()
	tconn, state, err := thx.TLSHandshaker.Handshake(ctx, conn, config)
	finished := time.Since(thx.begin).Seconds()
	thx.db.InsertIntoTLSHandshake(&QUICTLSHandshakeEvent{
		Network:         network,
		RemoteAddr:      remoteAddr,
		SNI:             config.ServerName,
		ALPN:            config.NextProtos,
		SkipVerify:      config.InsecureSkipVerify,
		Started:         started,
		Finished:        finished,
		Failure:         NewFailure(err),
		TLSVersion:      netxlite.TLSVersionString(state.Version),
		CipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		NegotiatedProto: state.NegotiatedProtocol,
		PeerCerts:       peerCerts(err, &state),
	})
	return tconn, state, err
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
