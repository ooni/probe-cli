package dialer

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/tlsx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// SaverDialer saves events occurring during the dial
type SaverDialer struct {
	Dialer
	Saver *trace.Saver
}

// DialContext implements Dialer.DialContext
func (d SaverDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	stop := time.Now()
	d.Saver.Write(trace.Event{
		Address:  address,
		Duration: stop.Sub(start),
		Err:      err,
		Name:     errorx.ConnectOperation,
		Proto:    network,
		Time:     stop,
	})
	return conn, err
}

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

// SaverConnDialer wraps the returned connection such that we
// collect all the read/write events that occur.
type SaverConnDialer struct {
	Dialer
	Saver *trace.Saver
}

// DialContext implements Dialer.DialContext
func (d SaverConnDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return saverConn{saver: d.Saver, Conn: conn}, nil
}

type saverConn struct {
	net.Conn
	saver *trace.Saver
}

func (c saverConn) Read(p []byte) (int, error) {
	start := time.Now()
	count, err := c.Conn.Read(p)
	stop := time.Now()
	c.saver.Write(trace.Event{
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     errorx.ReadOperation,
		Time:     stop,
	})
	return count, err
}

func (c saverConn) Write(p []byte) (int, error) {
	start := time.Now()
	count, err := c.Conn.Write(p)
	stop := time.Now()
	c.saver.Write(trace.Event{
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     errorx.WriteOperation,
		Time:     stop,
	})
	return count, err
}

var _ Dialer = SaverDialer{}
var _ TLSHandshaker = SaverTLSHandshaker{}
var _ net.Conn = saverConn{}
