package netplumbing

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/bassosimone/quic-go"
)

// QUICHandshaker performs the QUIC handshake.
type QUICHandshaker interface {
	QUICHandshake(ctx context.Context, pconn net.PacketConn, remoteAddr net.Addr,
		tlsConf *tls.Config, config *quic.Config) (quic.EarlySession, error)
}

// quicGoHandshaker implements QUICHandshaker using quic-go
type quicGoHandshaker struct{}

// QUICHandshake implements QUICHandshaker.QUICHandshake.
func (*quicGoHandshaker) QUICHandshake(ctx context.Context, pconn net.PacketConn,
	remoteAddr net.Addr, tlsConf *tls.Config, config *quic.Config) (
	quic.EarlySession, error) {
	return quic.DialEarlyContext(ctx, pconn, remoteAddr, "", tlsConf, config)
}

// ErrQUICHandshake is an error during the QUIC handshake.
type ErrQUICHandshake struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrQUICHandshake) Unwrap() error {
	return err.error
}

// quicHandshake performs the QUIC handshake.
func (txp *Transport) quicHandshake(
	ctx context.Context, network, ipaddr, port, sni string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	// TODO(bassosimone): allow to override the SNI
	// TODO(bassosimone): implement this part
	//if tlsConfig.RootCAs == nil {
	//}
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = sni
	}
	// TODO(bassosimone): ovverride fields of quic config?
	log := txp.logger(ctx)
	epnt := net.JoinHostPort(ipaddr, port)
	prefix := fmt.Sprintf("quicHandshake: %s/%s sni=%s alpn=%s...",
		epnt, network, sni, tlsConfig.NextProtos)
	log.Debug(prefix)
	sess, err := txp.doQUICHandshake(ctx, network, ipaddr, port, tlsConfig, quicConfig)
	if err != nil {
		log.Debugf("%s %s", prefix, err.Error())
		return nil, &ErrQUICHandshake{err}
	}
	log.Debugf("%s ok", prefix)
	return sess, nil
}

func (txp *Transport) doQUICHandshake(ctx context.Context, network, ipaddr, sport string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	port, err := strconv.Atoi(sport)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(ipaddr)
	if ip == nil {
		// TODO(kelmenhorst): write test for this error condition.
		return nil, errors.New("netplumbing: invalid IP representation")
	}
	conn, err := txp.quicListen(ctx)
	if err != nil {
		return nil, err
	}
	udpAddr := &net.UDPAddr{IP: ip, Port: port, Zone: ""}
	var qh QUICHandshaker = &quicGoHandshaker{}
	if config := ContextConfig(ctx); config != nil && config.QUICHandshaker != nil {
		qh = config.QUICHandshaker
	}
	return qh.QUICHandshake(ctx, conn, udpAddr, tlsConfig, quicConfig)
}
