package quicdialer

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// HandshakeSaver saves events occurring during the handshake
type HandshakeSaver struct {
	Saver  *trace.Saver
	Dialer ContextDialer
}

// DialContext implements ContextDialer.DialContext
func (h HandshakeSaver) DialContext(ctx context.Context, network string,
	host string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	start := time.Now()
	// TODO(bassosimone): in the future we probably want to also save
	// information about what versions we're willing to accept.
	h.Saver.Write(trace.Event{
		Address:       host,
		Name:          "quic_handshake_start",
		NoTLSVerify:   tlsCfg.InsecureSkipVerify,
		Proto:         network,
		TLSNextProtos: tlsCfg.NextProtos,
		TLSServerName: tlsCfg.ServerName,
		Time:          start,
	})
	sess, err := h.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	stop := time.Now()
	if err != nil {
		h.Saver.Write(trace.Event{
			Duration:      stop.Sub(start),
			Err:           err,
			Name:          "quic_handshake_done",
			NoTLSVerify:   tlsCfg.InsecureSkipVerify,
			TLSNextProtos: tlsCfg.NextProtos,
			TLSServerName: tlsCfg.ServerName,
			Time:          stop,
		})
		return nil, err
	}
	state := ConnectionState(sess)
	h.Saver.Write(trace.Event{
		Duration:           stop.Sub(start),
		Name:               "quic_handshake_done",
		NoTLSVerify:        tlsCfg.InsecureSkipVerify,
		TLSCipherSuite:     tlsx.CipherSuiteString(state.CipherSuite),
		TLSNegotiatedProto: state.NegotiatedProtocol,
		TLSNextProtos:      tlsCfg.NextProtos,
		TLSPeerCerts:       trace.PeerCerts(state, err),
		TLSServerName:      tlsCfg.ServerName,
		TLSVersion:         tlsx.VersionString(state.Version),
		Time:               stop,
	})
	return sess, nil
}
