package archival

//
// Saves TLS events
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

// QUICTLSHandshakeEvent contains a QUIC or TLS handshake event.
type QUICTLSHandshakeEvent struct {
	ALPN            []string
	CipherSuite     string
	Failure         error
	Finished        time.Time
	NegotiatedProto string
	Network         string
	PeerCerts       [][]byte
	RemoteAddr      string
	SNI             string
	SkipVerify      bool
	Started         time.Time
	TLSVersion      string
}

// TLSHandshake performs a TLS handshake with the given handshaker
// and saves the results into the saver.
func (s *Saver) TLSHandshake(ctx context.Context, thx model.TLSHandshaker,
	conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
	network := conn.RemoteAddr().Network()
	remoteAddr := conn.RemoteAddr().String()
	started := time.Now()
	tconn, state, err := thx.Handshake(ctx, conn, config)
	// Implementation note: state is an empty ConnectionState on failure
	// so it's safe to access its fields also in that case
	s.appendTLSHandshake(&QUICTLSHandshakeEvent{
		ALPN:            config.NextProtos,
		CipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		Failure:         err,
		Finished:        time.Now(),
		NegotiatedProto: state.NegotiatedProtocol,
		Network:         network,
		PeerCerts:       s.tlsPeerCerts(err, &state),
		RemoteAddr:      remoteAddr,
		SNI:             config.ServerName,
		SkipVerify:      config.InsecureSkipVerify,
		Started:         started,
		TLSVersion:      netxlite.TLSVersionString(state.Version),
	})
	return tconn, state, err
}

func (s *Saver) appendTLSHandshake(ev *QUICTLSHandshakeEvent) {
	s.mu.Lock()
	s.trace.TLSHandshake = append(s.trace.TLSHandshake, ev)
	s.mu.Unlock()
}

func (s *Saver) tlsPeerCerts(err error, state *tls.ConnectionState) (out [][]byte) {
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

// WrapTLSHandshaker takes in input a TLS handshaker and returns
// a new one that uses this saver for saving events.
func (s *Saver) WrapTLSHandshaker(th model.TLSHandshaker) model.TLSHandshaker {
	return &tlsHandshakerSaver{TLSHandshaker: th, s: s}
}

type tlsHandshakerSaver struct {
	model.TLSHandshaker
	s *Saver
}

func (th *tlsHandshakerSaver) Handshake(ctx context.Context,
	conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
	return th.s.TLSHandshake(ctx, th.TLSHandshaker, conn, config)
}
