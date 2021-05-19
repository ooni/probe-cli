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

// http3dial is the top-level function called by the transport
// when we need to establish a new http3 connection.
func (txp *Transport) http3dial(ctx context.Context, network string,
	address string, tlsConfig *tls.Config, quicConfig *quic.Config) (
	quic.EarlySession, error) {
	return txp.http3dialWrapError(ctx, network, address, tlsConfig, quicConfig)
}

// http3dialWrapError wraps errors using ErrQUICDial.
func (txp *Transport) http3dialWrapError(
	ctx context.Context, network, address string, tlsConfig *tls.Config,
	quicConfig *quic.Config) (quic.EarlySession, error) {
	sess, err := txp.http3dialRejectProxy(ctx, network, address, tlsConfig, quicConfig)
	if err != nil {
		return nil, &ErrQUICDial{err}
	}
	return sess, nil
}

// ErrQUICDial is an error occurred when dialing a QUIC connection.
type ErrQUICDial struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrQUICDial) Unwrap() error {
	return err.error
}

// http3dialRejectProxy clarifies that we cannot run with a tcp proxy.
func (txp *Transport) http3dialRejectProxy(
	ctx context.Context, network, address string, tlsConfig *tls.Config,
	quicConfig *quic.Config) (quic.EarlySession, error) {
	if config := ContextConfig(ctx); config != nil && config.Proxy != nil {
		return nil, fmt.Errorf("%w when using QUIC", ErrProxyNotImplemented)
	}
	return txp.http3dialEmitLogs(ctx, network, address, tlsConfig, quicConfig)
}

// http3dialEmitLogs emits the http3dial logs.
func (txp *Transport) http3dialEmitLogs(
	ctx context.Context, network, address string, tlsConfig *tls.Config,
	quicConfig *quic.Config) (quic.EarlySession, error) {
	log := txp.logger(ctx)
	log.Debugf("quicDial: %s/%s...", address, network)
	sess, err := txp.http3DialResolveAndLoop(
		ctx, network, address, tlsConfig, quicConfig)
	if err != nil {
		log.Debugf("quicDial: %s/%s... %s", address, network, err)
		return nil, err
	}
	log.Debugf("quicDial: %s/%s... ok", address, network)
	return sess, nil
}

// http3DialResolveAndLoop resolves the domain name in address
// to IP addresses, and tries every address until one of them
// succeeds or all of them have failed.
func (txp *Transport) http3DialResolveAndLoop(
	ctx context.Context, network, address string, tlsConfig *tls.Config,
	quicConfig *quic.Config) (quic.EarlySession, error) {
	hostname, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ipaddrs, err := txp.LookupHost(ctx, hostname)
	if err != nil {
		return nil, err
	}
	aggregate := &ErrAllHandshakesFailed{}
	for _, ipaddr := range ipaddrs {
		sess, err := txp.quicHandshake(ctx, network, ipaddr, port, hostname,
			tlsConfig, quicConfig)
		if err == nil {
			return sess, nil
		}
		aggregate.Errors = append(aggregate.Errors, err)
	}
	return nil, aggregate
}

// ErrAllHandshakesFailed indicates that all QUIC handshakes failed.
type ErrAllHandshakesFailed struct {
	// Errors contains all the errors that occurred.
	Errors []error
}

// Error implements error.Error.
func (err *ErrAllHandshakesFailed) Error() string {
	return fmt.Sprintf("one or more quic handshakes failed: %#v", err.Errors)
}

// quicHandshake is the top-level entry point for performing a QUIC handshake.
func (txp *Transport) quicHandshake(
	ctx context.Context, network, ipaddr, port, sni string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	return txp.quicHandshakeWrapError(
		ctx, network, ipaddr, port, sni, tlsConfig, quicConfig)
}

// quicHandshakeWrapError wraps errors using ErrQUICHandshake.
func (txp *Transport) quicHandshakeWrapError(
	ctx context.Context, network, ipaddr, port, sni string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	sess, err := txp.quicHandshakePatchConfig(
		ctx, network, ipaddr, port, sni, tlsConfig, quicConfig)
	if err != nil {
		return nil, &ErrQUICHandshake{err}
	}
	return sess, nil
}

// ErrQUICHandshake is an error during the QUIC handshake.
type ErrQUICHandshake struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrQUICHandshake) Unwrap() error {
	return err.error
}

// quicHandshakePatchConfig patches the config we're passing to the real
// handshaker taking into accoint the value of overrides.
func (txp *Transport) quicHandshakePatchConfig(
	ctx context.Context, network, ipaddr, port, sni string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	if config := ContextConfig(ctx); config != nil && config.TLSClientConfig != nil {
		if config.TLSClientConfig.ServerName != "" {
			tlsConfig.ServerName = config.TLSClientConfig.ServerName
		}
		if len(config.TLSClientConfig.NextProtos) > 0 {
			tlsConfig.NextProtos = config.TLSClientConfig.NextProtos
		}
	}
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = sni
	}
	// TODO(bassosimone): implement this part
	//if tlsConfig.RootCAs == nil {
	//}
	if config := ContextConfig(ctx); config != nil && config.QUICConfig != nil {
		if len(config.QUICConfig.Versions) > 0 {
			quicConfig.Versions = config.QUICConfig.Versions
		}
	}
	return txp.quicHandshakeEmitLogs(
		ctx, network, ipaddr, port, sni, tlsConfig, quicConfig)
}

// quicHandshakeEmitLogs emits the QUIC handshake logs.
func (txp *Transport) quicHandshakeEmitLogs(
	ctx context.Context, network, ipaddr, port, sni string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	log := txp.logger(ctx)
	epnt := net.JoinHostPort(ipaddr, port)
	prefix := fmt.Sprintf("quicHandshake: %s/%s sni=%s alpn=%s v=%+v...",
		epnt, network, sni, tlsConfig.NextProtos, quicConfig.Versions)
	log.Debug(prefix)
	sess, err := txp.quicHandshakeDoHandshake(
		ctx, network, ipaddr, port, tlsConfig, quicConfig)
	if err != nil {
		log.Debugf("%s %s", prefix, err.Error())
		return nil, err
	}
	log.Debugf("%s ok", prefix)
	return sess, nil
}

// quicHandshakeDoHandshake implements the QUIC handshake.
func (txp *Transport) quicHandshakeDoHandshake(
	ctx context.Context, network, ipaddr, sport string,
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

// quicListen creates a new listening UDP connection for QUIC.
func (txp *Transport) quicListen(ctx context.Context) (net.PacketConn, error) {
	ql := txp.DefaultQUICListener()
	if config := ContextConfig(ctx); config != nil && config.QUICListener != nil {
		ql = config.QUICListener
	}
	log := txp.logger(ctx)
	log.Debug("quic: start listening...")
	conn, err := ql.QUICListen(ctx)
	if err != nil {
		log.Debugf("quic: start listening... %s", err)
		return nil, &ErrQUICListen{err}
	}
	log.Debugf("quic: start listening... %s", conn.LocalAddr().String())
	return &quicUDPConnWrapper{
		byteCounter: txp.byteCounter(ctx), UDPConn: conn}, nil
}

// ErrQUICListen is a listen error.
type ErrQUICListen struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrQUICListen) Unwrap() error {
	return err.error
}

// quicGoHandshaker implements QUICHandshaker using quic-go
type quicGoHandshaker struct{}

// QUICHandshake implements QUICHandshaker.QUICHandshake.
func (*quicGoHandshaker) QUICHandshake(ctx context.Context, pconn net.PacketConn,
	remoteAddr net.Addr, tlsConf *tls.Config, config *quic.Config) (
	quic.EarlySession, error) {
	return quic.DialEarlyContext(ctx, pconn, remoteAddr, "", tlsConf, config)
}

// DefaultQUICListener returns the default QUICListener.
func (txp *Transport) DefaultQUICListener() QUICListener {
	return &quicStdlibListener{}
}

// quicStdlibListener is a QUICListener using the Go stdlib.
type quicStdlibListener struct{}

// QUICListen implements QUICListener.QUICListen.
func (ql *quicStdlibListener) QUICListen(ctx context.Context) (*net.UDPConn, error) {
	return net.ListenUDP("udp", &net.UDPAddr{})
}

// quicUDPConnWrapper wraps an udpConn connection used by QUIC.
type quicUDPConnWrapper struct {
	byteCounter ByteCounter
	*net.UDPConn
}

// ReadMsgUDP reads a message from an UDP socket.
func (conn *quicUDPConnWrapper) ReadMsgUDP(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
	n, oobn, flags, addr, err := conn.UDPConn.ReadMsgUDP(b, oob)
	if err != nil {
		return 0, 0, 0, nil, &ErrReadFrom{err}
	}
	conn.byteCounter.CountBytesReceived(n + oobn)
	return n, oobn, flags, addr, nil
}

// ErrReadFrom is a readFrom error.
type ErrReadFrom struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrReadFrom) Unwrap() error {
	return err.error
}

// WriteTo writes a message to the UDP socket.
func (conn *quicUDPConnWrapper) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := conn.UDPConn.WriteTo(p, addr)
	if err != nil {
		return 0, &ErrWriteTo{err}
	}
	conn.byteCounter.CountBytesSent(count)
	return count, nil
}

// ErrWriteTo is a writeTo error.
type ErrWriteTo struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrWriteTo) Unwrap() error {
	return err.error
}
