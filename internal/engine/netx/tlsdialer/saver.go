package tlsdialer

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// SaverTLSHandshaker saves events occurring during the handshake
type SaverTLSHandshaker struct {
	TLSHandshaker
	Saver *trace.Saver
}

// Handshake implements TLSHandshaker.Handshake
func (h SaverTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	start := time.Now()
	h.Saver.Write(trace.Event{
		Name:          "tls_handshake_start",
		NoTLSVerify:   config.InsecureSkipVerify,
		TLSNextProtos: config.NextProtos,
		TLSServerName: config.ServerName,
		Time:          start,
	})
	tlsconn, state, err := h.TLSHandshaker.Handshake(ctx, conn, config)
	stop := time.Now()
	h.Saver.Write(trace.Event{
		Duration:           stop.Sub(start),
		Err:                err,
		Name:               "tls_handshake_done",
		NoTLSVerify:        config.InsecureSkipVerify,
		TLSCipherSuite:     tlsx.CipherSuiteString(state.CipherSuite),
		TLSNegotiatedProto: state.NegotiatedProtocol,
		TLSNextProtos:      config.NextProtos,
		TLSPeerCerts:       trace.PeerCerts(state, err),
		TLSServerName:      config.ServerName,
		TLSVersion:         tlsx.VersionString(state.Version),
		Time:               stop,
	})
	return tlsconn, state, err
}

var _ TLSHandshaker = SaverTLSHandshaker{}
