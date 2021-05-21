package netplumbing

// This file contains the implementation of Transport.http3dial.

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

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
	sess, err := txp.quicHandshakeListenAndHanshake(
		ctx, network, ipaddr, port, tlsConfig, quicConfig)
	if err != nil {
		log.Debugf("%s %s", prefix, err.Error())
		return nil, err
	}
	log.Debugf("%s ok", prefix)
	return sess, nil
}

// quicHandshakeListenAndHanshake listens and starts the handshake.
func (txp *Transport) quicHandshakeListenAndHanshake(
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
	return txp.quicHandshakeMaybeTrace(ctx, conn, udpAddr, tlsConfig, quicConfig)
}

// quicHandshakeMaybeTrace enables tracing if needed.
func (txp *Transport) quicHandshakeMaybeTrace(
	ctx context.Context, conn net.PacketConn, udpAddr *net.UDPAddr,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	if th := ContextTraceHeader(ctx); th != nil {
		return txp.quicHandshakeWithTraceHeader(
			ctx, conn, udpAddr, tlsConfig, quicConfig, th)
	}
	return txp.quicHandshakeMaybeOverride(ctx, conn, udpAddr, tlsConfig, quicConfig)
}

// quicHandshakeWithTraceHeader traces the QUIC handshake
func (txp *Transport) quicHandshakeWithTraceHeader(
	ctx context.Context, conn net.PacketConn, udpAddr *net.UDPAddr,
	tlsConfig *tls.Config, quicConfig *quic.Config, th *TraceHeader) (
	quic.EarlySession, error) {
	ev := &TLSHandshakeTrace{
		kind:          TraceKindQUICHandshake,
		LocalAddr:     conn.LocalAddr().String(),
		RemoteAddr:    udpAddr.String(),
		SkipTLSVerify: tlsConfig.InsecureSkipVerify,
		NextProtos:    tlsConfig.NextProtos,
		StartTime:     time.Now(),
		Error:         nil,
	}
	if net.ParseIP(tlsConfig.ServerName) == nil {
		ev.ServerName = tlsConfig.ServerName
	}
	defer th.add(ev)
	sess, err := txp.quicHandshakeMaybeOverride(
		ctx, conn, udpAddr, tlsConfig, quicConfig)
	ev.EndTime = time.Now()
	ev.Error = err
	if err != nil {
		return nil, err
	}
	state := sess.ConnectionState()
	ev.Version = state.Version
	ev.CipherSuite = state.CipherSuite
	ev.NegotiatedProto = state.NegotiatedProtocol
	for _, c := range state.PeerCertificates {
		ev.PeerCerts = append(ev.PeerCerts, c.Raw)
	}
	return sess, nil
}

// quicHandshakeMaybeOverride calls the default or the override handshaker.
func (txp *Transport) quicHandshakeMaybeOverride(
	ctx context.Context, conn net.PacketConn, udpAddr *net.UDPAddr,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	qh := txp.DefaultQUICHandshaker()
	if config := ContextConfig(ctx); config != nil && config.QUICHandshaker != nil {
		qh = config.QUICHandshaker
	}
	return qh.QUICHandshake(ctx, conn, udpAddr, tlsConfig, quicConfig)
}

// quicGoHandshaker implements QUICHandshaker using quic-go
type quicGoHandshaker struct{}

// QUICHandshake implements QUICHandshaker.QUICHandshake.
func (*quicGoHandshaker) QUICHandshake(ctx context.Context, pconn net.PacketConn,
	remoteAddr net.Addr, tlsConf *tls.Config, config *quic.Config) (
	quic.EarlySession, error) {
	return quic.DialEarlyContext(ctx, pconn, remoteAddr, "", tlsConf, config)
}

// DefaultQUICHandshaker returns the QUIC handshaker used by default.
func (txp *Transport) DefaultQUICHandshaker() QUICHandshaker {
	return &quicGoHandshaker{}
}

// quicListen is the top-level entry for creating a listening connection.
func (txp *Transport) quicListen(ctx context.Context) (net.PacketConn, error) {
	return txp.quicListenWrapError(ctx)
}

// quicListenWrapError wraps the returned error as a QUICListenError.
func (txp *Transport) quicListenWrapError(ctx context.Context) (net.PacketConn, error) {
	conn, err := txp.quicListenEmitLogs(ctx)
	if err != nil {
		return nil, &ErrQUICListen{err}
	}
	return conn, nil
}

// ErrQUICListen is a listen error.
type ErrQUICListen struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrQUICListen) Unwrap() error {
	return err.error
}

// quicListenEmitLogs emits QUIC listen logs.
func (txp *Transport) quicListenEmitLogs(ctx context.Context) (net.PacketConn, error) {
	log := txp.logger(ctx)
	log.Debug("quic: start listening...")
	conn, err := txp.quicListenWrapConn(ctx)
	if err != nil {
		log.Debugf("quic: start listening... %s", err)
		return nil, err
	}
	log.Debugf("quic: start listening... %s", conn.LocalAddr().String())
	return conn, nil
}

// quicListenWrapConn wraps the listening conn.
func (txp *Transport) quicListenWrapConn(ctx context.Context) (net.PacketConn, error) {
	conn, err := txp.quicListenMaybeTrace(ctx)
	if err != nil {
		return nil, err
	}
	return &quicUDPConnWrapper{
		byteCounter: txp.byteCounter(ctx),
		PacketConn:  conn,
	}, nil
}

// quicUDPConnWrapper wraps an udpConn connection used by QUIC.
type quicUDPConnWrapper struct {
	byteCounter ByteCounter
	net.PacketConn
}

// TODO(bassosimone): figure out why ReadMsgUDP is not called. Consider
// whether we care about that. It seems the code is WAI both using Linux
// and macOS. Therefore, it may be that we are not picking up the right
// interface implementing ReadMsgUDP because of wrapping.
//
// See https://pkg.go.dev/github.com/lucas-clemente/quic-go#OOBCapablePacketConn
// for the official documentation regarding ReadMsgUDP vs ReadFrom.

// ReadFrom reads a message from an UDP socket.
func (conn *quicUDPConnWrapper) ReadFrom(p []byte) (int, net.Addr, error) {
	n, addr, err := conn.PacketConn.ReadFrom(p)
	if err != nil {
		return 0, nil, &ErrReadFrom{err}
	}
	conn.byteCounter.CountBytesReceived(n)
	return n, addr, nil
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
	count, err := conn.PacketConn.WriteTo(p, addr)
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

// quicListenMaybeTrace activates traces if needed
func (txp *Transport) quicListenMaybeTrace(ctx context.Context) (net.PacketConn, error) {
	conn, err := txp.quicListenMaybeOverride(ctx)
	if err != nil {
		return nil, err
	}
	if th := ContextTraceHeader(ctx); th != nil {
		conn = &tracingQUICUDPConn{PacketConn: conn, th: th}
	}
	return conn, nil
}

// tracingQUICUDPConn adds tracing to a QUICUDPConn
type tracingQUICUDPConn struct {
	th *TraceHeader
	net.PacketConn
}

// ReadMsgUDP reads a message from an UDP socket.
func (conn *tracingQUICUDPConn) ReadFrom(p []byte) (int, net.Addr, error) {
	ev := &ReadWriteTrace{
		kind:       TraceKindReadFrom,
		LocalAddr:  conn.PacketConn.LocalAddr().String(),
		BufferSize: len(p),
		StartTime:  time.Now(),
	}
	defer conn.th.add(ev)
	n, addr, err := conn.PacketConn.ReadFrom(p)
	if addr != nil {
		ev.RemoteAddr = addr.String()
	}
	ev.EndTime = time.Now()
	ev.Count = n
	ev.Error = err
	return n, addr, err
}

// WriteTo writes a message to the UDP socket.
func (conn *tracingQUICUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	ev := &ReadWriteTrace{
		kind:       TraceKindWriteTo,
		LocalAddr:  conn.PacketConn.LocalAddr().String(),
		RemoteAddr: addr.String(),
		BufferSize: len(p),
		StartTime:  time.Now(),
	}
	defer conn.th.add(ev)
	count, err := conn.PacketConn.WriteTo(p, addr)
	ev.EndTime = time.Now()
	ev.Count = count
	ev.Error = err
	return count, err
}

// quicListenMaybeOverride uses the default or the overriden QUIC listener.
func (txp *Transport) quicListenMaybeOverride(ctx context.Context) (net.PacketConn, error) {
	ql := txp.DefaultQUICListener()
	if config := ContextConfig(ctx); config != nil && config.QUICListener != nil {
		ql = config.QUICListener
	}
	return ql.QUICListen(ctx)
}

// DefaultQUICListener returns the default QUICListener.
func (txp *Transport) DefaultQUICListener() QUICListener {
	return &quicStdlibListener{}
}

// quicStdlibListener is a QUICListener using the Go stdlib.
type quicStdlibListener struct{}

// QUICListen implements QUICListener.QUICListen.
func (ql *quicStdlibListener) QUICListen(ctx context.Context) (net.PacketConn, error) {
	return net.ListenUDP("udp", &net.UDPAddr{})
}
