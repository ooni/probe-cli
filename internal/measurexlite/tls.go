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
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// NewTLSHandshakerStdlib is equivalent to netxlite.NewTLSHandshakerStdlib
// except that it returns a model.TLSHandshaker that uses this trace.
func (tx *Trace) NewTLSHandshakerStdlib(dl model.DebugLogger) model.TLSHandshaker {
	return &tlsHandshakerTrace{
		thx: tx.newTLSHandshakerStdlib(dl),
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
	ctx context.Context, conn net.Conn, tlsConfig *tls.Config) (net.Conn, tls.ConnectionState, error) {
	return thx.thx.Handshake(netxlite.ContextWithTrace(ctx, thx.tx), conn, tlsConfig)
}

// OnTLSHandshakeStart implements model.Trace.OnTLSHandshakeStart.
func (tx *Trace) OnTLSHandshakeStart(now time.Time, remoteAddr string, config *tls.Config) {
	t := now.Sub(tx.ZeroTime)
	select {
	case tx.networkEvent <- NewAnnotationArchivalNetworkEvent(tx.Index, t, "tls_handshake_start"):
	default: // buffer is full
	}
}

// OnTLSHandshakeDone implements model.Trace.OnTLSHandshakeDone.
func (tx *Trace) OnTLSHandshakeDone(started time.Time, remoteAddr string, config *tls.Config,
	state tls.ConnectionState, err error, finished time.Time) {
	t := finished.Sub(tx.ZeroTime)
	select {
	case tx.tlsHandshake <- NewArchivalTLSOrQUICHandshakeResult(
		tx.Index,
		started.Sub(tx.ZeroTime),
		"tls",
		remoteAddr,
		config,
		state,
		err,
		t,
	):
	default: // buffer is full
	}
	select {
	case tx.networkEvent <- NewAnnotationArchivalNetworkEvent(tx.Index, t, "tls_handshake_done"):
	default: // buffer is full
	}
}

// NewArchivalTLSOrQUICHandshakeResult generates a model.ArchivalTLSOrQUICHandshakeResult
// from the available information right after the TLS handshake returns.
func NewArchivalTLSOrQUICHandshakeResult(
	index int64, started time.Duration, network string, address string, config *tls.Config,
	state tls.ConnectionState, err error, finished time.Duration) *model.ArchivalTLSOrQUICHandshakeResult {
	return &model.ArchivalTLSOrQUICHandshakeResult{
		Network:            network,
		Address:            address,
		CipherSuite:        netxlite.TLSCipherSuiteString(state.CipherSuite),
		Failure:            tracex.NewFailure(err),
		NegotiatedProtocol: state.NegotiatedProtocol,
		NoTLSVerify:        config.InsecureSkipVerify,
		PeerCertificates:   TLSPeerCerts(state, err),
		ServerName:         config.ServerName,
		T:                  finished.Seconds(),
		Tags:               []string{},
		TLSVersion:         netxlite.TLSVersionString(state.Version),
	}
}

// newArchivalBinaryData is a factory that adapts binary data to the
// model.ArchivalMaybeBinaryData format.
func newArchivalBinaryData(data []byte) model.ArchivalMaybeBinaryData {
	// TODO(https://github.com/ooni/probe/issues/2165): we should actually extend the
	// model's archival data format to have a pure-binary-data type for the cases in which
	// we know in advance we're dealing with binary data.
	return model.ArchivalMaybeBinaryData{
		Value: string(data),
	}
}

// TLSPeerCerts extracts the certificates either from the list of certificates
// in the connection state or from the error that occurred.
func TLSPeerCerts(
	state tls.ConnectionState, err error) (out []model.ArchivalMaybeBinaryData) {
	out = []model.ArchivalMaybeBinaryData{}
	var x509HostnameError x509.HostnameError
	if errors.As(err, &x509HostnameError) {
		// Test case: https://wrong.host.badssl.com/
		out = append(out, newArchivalBinaryData(x509HostnameError.Certificate.Raw))
		return
	}
	var x509UnknownAuthorityError x509.UnknownAuthorityError
	if errors.As(err, &x509UnknownAuthorityError) {
		// Test case: https://self-signed.badssl.com/. This error has
		// never been among the ones returned by MK.
		out = append(out, newArchivalBinaryData(x509UnknownAuthorityError.Cert.Raw))
		return
	}
	var x509CertificateInvalidError x509.CertificateInvalidError
	if errors.As(err, &x509CertificateInvalidError) {
		// Test case: https://expired.badssl.com/
		out = append(out, newArchivalBinaryData(x509CertificateInvalidError.Cert.Raw))
		return
	}
	for _, cert := range state.PeerCertificates {
		out = append(out, newArchivalBinaryData(cert.Raw))
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

// FirstTLSHandshake drains the network events buffered inside the TLSHandshake channel
// and returns the first TLSHandshake.
func (tx *Trace) FirstTLSHandshake() *model.ArchivalTLSOrQUICHandshakeResult {
	ev := tx.TLSHandshakes()
	if len(ev) < 1 {
		return nil
	}
	return ev[0]
}
