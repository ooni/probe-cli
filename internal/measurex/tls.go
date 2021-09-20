package measurex

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// TLSConn is the TLS conn type we use.
type TLSConn interface {
	netxlite.TLSConn

	// ConnID returns the connection ID.
	ConnID() int64
}

// TLSHandshaker is the TLS handshaker type we use. This handshaker
// will save TLS handshake events into the DB.
type TLSHandshaker interface {
	Handshake(ctx context.Context, conn Conn, config *tls.Config) (TLSConn, error)
}

// WrapTLSHandshaker wraps a netxlite.TLSHandshaker to return a new
// instance of TLSHandshaker that saves events into the DB.
func WrapTLSHandshaker(origin Origin, db EventDB, thx netxlite.TLSHandshaker) TLSHandshaker {
	return &tlsHandshakerx{TLSHandshaker: thx, db: db, origin: origin}
}

type tlsHandshakerx struct {
	netxlite.TLSHandshaker
	db     EventDB
	origin Origin
}

// TLSHandshakeEvent contains a TLS handshake event.
type TLSHandshakeEvent struct {
	Origin          Origin
	MeasurementID   int64
	ConnID          int64
	Engine          string
	Network         string
	RemoteAddr      string
	LocalAddr       string
	SNI             string
	ALPN            []string
	SkipVerify      bool
	Started         time.Duration
	Finished        time.Duration
	Error           error
	Oddity          Oddity
	TLSVersion      string
	CipherSuite     string
	NegotiatedProto string
	PeerCerts       [][]byte
}

func (thx *tlsHandshakerx) Handshake(ctx context.Context,
	conn Conn, config *tls.Config) (TLSConn, error) {
	network := conn.RemoteAddr().Network()
	remoteAddr := conn.RemoteAddr().String()
	localAddr := conn.LocalAddr().String()
	started := thx.db.ElapsedTime()
	tconn, state, err := thx.TLSHandshaker.Handshake(ctx, conn, config)
	finished := thx.db.ElapsedTime()
	thx.db.InsertIntoTLSHandshake(&TLSHandshakeEvent{
		Origin:          thx.origin,
		MeasurementID:   thx.db.MeasurementID(),
		ConnID:          conn.ConnID(),
		Engine:          "", // TODO(bassosimone): add support
		Network:         network,
		RemoteAddr:      remoteAddr,
		LocalAddr:       localAddr,
		SNI:             config.ServerName,
		ALPN:            config.NextProtos,
		SkipVerify:      config.InsecureSkipVerify,
		Started:         started,
		Finished:        finished,
		Error:           err,
		Oddity:          thx.computeOddity(err),
		TLSVersion:      netxlite.TLSVersionString(state.Version),
		CipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		NegotiatedProto: state.NegotiatedProtocol,
		PeerCerts:       peerCerts(err, &state),
	})
	if err != nil {
		return nil, err
	}
	return &tlsConnx{
		TLSConn: tconn.(netxlite.TLSConn), connID: conn.ConnID()}, nil
}

func (thx *tlsHandshakerx) computeOddity(err error) Oddity {
	if err == nil {
		return ""
	}
	switch err.Error() {
	case errorsx.FailureGenericTimeoutError:
		return OddityTLSHandshakeTimeout
	case errorsx.FailureConnectionReset:
		return OddityTLSHandshakeReset
	default:
		return OddityTLSHandshakeOther
	}
}

type tlsConnx struct {
	netxlite.TLSConn
	connID int64
}

func (c *tlsConnx) ConnID() int64 {
	return c.connID
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
