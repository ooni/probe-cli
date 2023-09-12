package tracex

//
// TLS
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

// TLSHandshakerSaver saves events occurring during the TLS handshake.
type TLSHandshakerSaver struct {
	// TLSHandshaker is the underlying TLS handshaker.
	TLSHandshaker model.TLSHandshaker

	// Saver is the saver in which to save events.
	Saver *Saver
}

// WrapTLSHandshaker wraps a model.TLSHandshaker with a SaverTLSHandshaker
// that will save the TLS handshake results into this Saver.
//
// When this function is invoked on a nil Saver, it will directly return
// the original TLSHandshaker without any wrapping.
func (s *Saver) WrapTLSHandshaker(thx model.TLSHandshaker) model.TLSHandshaker {
	if s == nil {
		return thx
	}
	return &TLSHandshakerSaver{
		TLSHandshaker: thx,
		Saver:         s,
	}
}

// Handshake implements model.TLSHandshaker.Handshake
func (h *TLSHandshakerSaver) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
	proto := conn.RemoteAddr().Network()
	remoteAddr := conn.RemoteAddr().String()
	start := time.Now()
	h.Saver.Write(&EventTLSHandshakeStart{&EventValue{
		Address:       remoteAddr,
		NoTLSVerify:   config.InsecureSkipVerify,
		Proto:         proto,
		TLSNextProtos: config.NextProtos,
		TLSServerName: config.ServerName,
		Time:          start,
	}})
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	stop := time.Now()
	h.Saver.Write(&EventTLSHandshakeDone{&EventValue{
		Address:            remoteAddr,
		Duration:           stop.Sub(start),
		Err:                NewFailureStr(err),
		NoTLSVerify:        config.InsecureSkipVerify,
		Proto:              proto,
		TLSCipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		TLSNegotiatedProto: state.NegotiatedProtocol,
		TLSNextProtos:      config.NextProtos,
		TLSPeerCerts:       tlsPeerCerts(state, err),
		TLSServerName:      config.ServerName,
		TLSVersion:         netxlite.TLSVersionString(state.Version),
		Time:               stop,
	}})
	return tlsconn, state, err
}

var _ model.TLSHandshaker = &TLSHandshakerSaver{}

// tlsPeerCerts returns the certificates presented by the peer regardless
// of whether the TLS handshake was successful
func tlsPeerCerts(state tls.ConnectionState, err error) (out [][]byte) {
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
