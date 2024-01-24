package measurexlite

//
// TLS tracing
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

// NewTLSHandshakerStdlib is equivalent to netxlite.Netx.NewTLSHandshakerStdlib
// except that it returns a model.TLSHandshaker that uses this trace.
func (tx *Trace) NewTLSHandshakerStdlib(dl model.DebugLogger) model.TLSHandshaker {
	return &tlsHandshakerTrace{
		thx: tx.Netx.NewTLSHandshakerStdlib(dl),
		tx:  tx,
	}
}

// tlsHandshakerTrace is a trace-aware TLS handshaker.
type tlsHandshakerTrace struct {
	thx model.TLSHandshaker
	tx  *Trace
}

var _ model.TLSHandshaker = &tlsHandshakerTrace{}

// Handshake implements model.TLSHandshaker.Handshake.
func (thx *tlsHandshakerTrace) Handshake(
	ctx context.Context, conn net.Conn, tlsConfig *tls.Config) (model.TLSConn, error) {
	return thx.thx.Handshake(netxlite.ContextWithTrace(ctx, thx.tx), conn, tlsConfig)
}

// OnTLSHandshakeStart implements model.Trace.OnTLSHandshakeStart.
func (tx *Trace) OnTLSHandshakeStart(now time.Time, remoteAddr string, config *tls.Config) {
	t := now.Sub(tx.ZeroTime())
	select {
	case tx.networkEvent <- NewAnnotationArchivalNetworkEvent(
		tx.Index(), t, "tls_handshake_start", tx.tags...):
	default: // buffer is full
	}
}

// OnTLSHandshakeDone implements model.Trace.OnTLSHandshakeDone.
func (tx *Trace) OnTLSHandshakeDone(started time.Time, remoteAddr string, config *tls.Config,
	state tls.ConnectionState, err error, finished time.Time) {
	t := finished.Sub(tx.ZeroTime())

	select {
	case tx.tlsHandshake <- NewArchivalTLSOrQUICHandshakeResult(
		tx.Index(),
		started.Sub(tx.ZeroTime()),
		"tcp",
		remoteAddr,
		config,
		state,
		err,
		t,
		tx.tags...,
	):
	default: // buffer is full
	}

	select {
	case tx.networkEvent <- NewAnnotationArchivalNetworkEvent(
		tx.Index(), t, "tls_handshake_done", tx.tags...):
	default: // buffer is full
	}
}

// NewArchivalTLSOrQUICHandshakeResult generates a model.ArchivalTLSOrQUICHandshakeResult
// from the available information right after the TLS handshake returns.
func NewArchivalTLSOrQUICHandshakeResult(
	index int64, started time.Duration, network string, address string, config *tls.Config,
	state tls.ConnectionState, err error, finished time.Duration,
	tags ...string) *model.ArchivalTLSOrQUICHandshakeResult {
	return &model.ArchivalTLSOrQUICHandshakeResult{
		Network:            network,
		Address:            address,
		CipherSuite:        netxlite.TLSCipherSuiteString(state.CipherSuite),
		Failure:            NewFailure(err),
		NegotiatedProtocol: state.NegotiatedProtocol,
		NoTLSVerify:        config.InsecureSkipVerify,
		PeerCertificates:   TLSPeerCerts(state, err),
		ServerName:         config.ServerName,
		T0:                 started.Seconds(),
		T:                  finished.Seconds(),
		Tags:               copyAndNormalizeTags(tags),
		TLSVersion:         netxlite.TLSVersionString(state.Version),
		TransactionID:      index,
	}
}

// TLSPeerCerts extracts the certificates either from the list of certificates
// in the connection state or from the error that occurred.
func TLSPeerCerts(
	state tls.ConnectionState, err error) (out []model.ArchivalBinaryData) {
	out = []model.ArchivalBinaryData{}

	var x509HostnameError x509.HostnameError
	if errors.As(err, &x509HostnameError) {
		// Test case: https://wrong.host.badssl.com/
		out = append(out, model.ArchivalBinaryData(x509HostnameError.Certificate.Raw))
		return
	}

	var x509UnknownAuthorityError x509.UnknownAuthorityError
	if errors.As(err, &x509UnknownAuthorityError) {
		// Test case: https://self-signed.badssl.com/. This error has
		// never been among the ones returned by MK.
		out = append(out, model.ArchivalBinaryData(x509UnknownAuthorityError.Cert.Raw))
		return
	}

	var x509CertificateInvalidError x509.CertificateInvalidError
	if errors.As(err, &x509CertificateInvalidError) {
		// Test case: https://expired.badssl.com/
		out = append(out, model.ArchivalBinaryData(x509CertificateInvalidError.Cert.Raw))
		return
	}

	for _, cert := range state.PeerCertificates {
		out = append(out, model.ArchivalBinaryData(cert.Raw))
	}
	return
}

// TLSHandshakes drains the network events buffered inside the TLSHandshake channel.
func (tx *Trace) TLSHandshakes() (out []*model.ArchivalTLSOrQUICHandshakeResult) {
	for {
		select {
		case ev := <-tx.tlsHandshake:
			out = append(out, ev)
		default:
			return // done
		}
	}
}

// FirstTLSHandshakeOrNil drains the network events buffered inside the TLSHandshake channel
// and returns the first TLSHandshake, if any. Otherwise, it returns nil.
func (tx *Trace) FirstTLSHandshakeOrNil() *model.ArchivalTLSOrQUICHandshakeResult {
	ev := tx.TLSHandshakes()
	if len(ev) < 1 {
		return nil
	}
	return ev[0]
}
