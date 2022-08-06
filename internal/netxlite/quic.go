package netxlite

//
// QUIC implementation
//

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"sync"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewQUICListener creates a new QUICListener using the standard
// library to create listening UDP sockets.
func NewQUICListener() model.QUICListener {
	return &quicListenerErrWrapper{&quicListenerStdlib{}}
}

// quicListenerStdlib is a QUICListener using the standard library.
type quicListenerStdlib struct{}

var _ model.QUICListener = &quicListenerStdlib{}

// Listen implements QUICListener.Listen.
func (qls *quicListenerStdlib) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return TProxy.ListenUDP("udp", addr)
}

// NewQUICDialerWithResolver is the WrapDialer equivalent for QUIC where
// we return a composed QUICDialer modified by optional wrappers.
//
// Please, note that this fuunction will just ignore any nil wrapper.
//
// Unlike the dialer returned by WrapDialer, this dialer MAY attempt
// happy eyeballs, perform parallel dial attempts, and return an error
// that aggregates all the errors that occurred.
func NewQUICDialerWithResolver(listener model.QUICListener, logger model.DebugLogger,
	resolver model.Resolver, wrappers ...model.QUICDialerWrapper) (outDialer model.QUICDialer) {
	outDialer = &quicDialerErrWrapper{
		QUICDialer: &quicDialerHandshakeCompleter{
			Dialer: &quicDialerQUICGo{
				QUICListener: listener,
			},
		},
	}
	for _, wrapper := range wrappers {
		if wrapper == nil {
			continue // ignore as documented
		}
		outDialer = wrapper.WrapQUICDialer(outDialer) // extend with user-supplied constructors
	}
	return &quicDialerLogger{
		Dialer: &quicDialerResolver{
			Dialer: &quicDialerLogger{
				Dialer:          outDialer,
				Logger:          logger,
				operationSuffix: "_address",
			},
			Resolver: resolver,
		},
		Logger: logger,
	}
}

// NewQUICDialerWithoutResolver is equivalent to calling NewQUICDialerWithResolver
// with the resolver argument set to &NullResolver{}.
func NewQUICDialerWithoutResolver(listener model.QUICListener,
	logger model.DebugLogger, wrappers ...model.QUICDialerWrapper) model.QUICDialer {
	return NewQUICDialerWithResolver(listener, logger, &NullResolver{}, wrappers...)
}

// quicDialerQUICGo dials using the lucas-clemente/quic-go library.
type quicDialerQUICGo struct {
	// QUICListener is the underlying QUICListener to use.
	QUICListener model.QUICListener

	// mockDialEarlyContext allows to mock quic.DialEarlyContext.
	mockDialEarlyContext func(ctx context.Context, pconn net.PacketConn,
		remoteAddr net.Addr, host string, tlsConfig *tls.Config,
		quicConfig *quic.Config) (quic.EarlyConnection, error)
}

var _ model.QUICDialer = &quicDialerQUICGo{}

// ErrInvalidIP indicates that a string is not a valid IP.
var ErrInvalidIP = errors.New("netxlite: invalid IP")

// ParseUDPAddr maps the string representation of an UDP endpoint to the
// corresponding *net.UDPAddr representation.
func ParseUDPAddr(address string) (*net.UDPAddr, error) {
	addr, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ipAddr := net.ParseIP(addr)
	if ipAddr == nil {
		return nil, ErrInvalidIP
	}
	dport, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	udpAddr := &net.UDPAddr{
		IP:   ipAddr,
		Port: dport,
		Zone: "",
	}
	return udpAddr, nil
}

// DialContext implements QUICDialer.DialContext. This function will
// apply the following TLS defaults:
//
// 1. if tlsConfig.RootCAs is nil, we use the Mozilla CA that we
// bundle with this measurement library;
//
// 2. if tlsConfig.NextProtos is empty _and_ the port is 443 or 8853,
// then we configure, respectively, "h3" and "dq".
func (d *quicDialerQUICGo) DialContext(ctx context.Context, network string,
	address string, tlsConfig *tls.Config, quicConfig *quic.Config) (
	quic.EarlyConnection, error) {
	udpAddr, err := ParseUDPAddr(address)
	if err != nil {
		return nil, err
	}
	pconn, err := d.QUICListener.Listen(&net.UDPAddr{IP: net.IPv4zero, Port: 0, Zone: ""})
	if err != nil {
		return nil, err
	}
	tlsConfig = d.maybeApplyTLSDefaults(tlsConfig, udpAddr.Port)
	trace := ContextTraceOrDefault(ctx)
	started := trace.TimeNow()
	trace.OnQUICHandshakeStart(started, address, quicConfig)
	qconn, err := d.dialEarlyContext(
		ctx, pconn, udpAddr, address, tlsConfig, quicConfig)
	finished := trace.TimeNow()
	trace.OnQUICHandshakeDone(started, address, qconn, tlsConfig, err, finished)
	if err != nil {
		pconn.Close() // we own it on failure
		return nil, err
	}
	return newQUICConnectionOwnsConn(qconn, pconn), nil
}

func (d *quicDialerQUICGo) dialEarlyContext(ctx context.Context,
	pconn net.PacketConn, remoteAddr net.Addr, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
	if d.mockDialEarlyContext != nil {
		return d.mockDialEarlyContext(
			ctx, pconn, remoteAddr, address, tlsConfig, quicConfig)
	}
	return quic.DialEarlyContext(
		ctx, pconn, remoteAddr, address, tlsConfig, quicConfig)
}

// maybeApplyTLSDefaults ensures that we're using our certificate pool, if
// needed, and that we use a suitable ALPN, if needed, for h3 and dq.
func (d *quicDialerQUICGo) maybeApplyTLSDefaults(config *tls.Config, port int) *tls.Config {
	config = config.Clone()
	if config.RootCAs == nil {
		config.RootCAs = defaultCertPool
	}
	if len(config.NextProtos) <= 0 {
		switch port {
		case 443:
			config.NextProtos = []string{"h3"}
		case 8853:
			// See https://datatracker.ietf.org/doc/html/draft-ietf-dprive-dnsoquic-02#section-10
			config.NextProtos = []string{"dq"}
		}
	}
	return config
}

// CloseIdleConnections closes idle connections.
func (d *quicDialerQUICGo) CloseIdleConnections() {
	// nothing to do
}

// quicDialerHandshakeCompleter ensures we complete the handshake.
type quicDialerHandshakeCompleter struct {
	Dialer model.QUICDialer
}

var _ model.QUICDialer = &quicDialerHandshakeCompleter{}

// DialContext implements model.QUICDialer.DialContext.
func (d *quicDialerHandshakeCompleter) DialContext(
	ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address, tlsConfig, quicConfig)
	if err != nil {
		return nil, err
	}
	select {
	case <-conn.HandshakeComplete().Done():
		return conn, nil
	case <-ctx.Done():
		conn.CloseWithError(0, "") // we own the conn
		return nil, ctx.Err()
	}
}

// CloseIdleConnections implements model.QUICDialer.CloseIdleConnections.
func (d *quicDialerHandshakeCompleter) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// quicConnectionOwnsConn ensures that we close the UDPLikeConn.
type quicConnectionOwnsConn struct {
	// EarlyConnection is the embedded early connection
	quic.EarlyConnection

	// conn is the connection we own
	conn model.UDPLikeConn
}

func newQUICConnectionOwnsConn(qconn quic.EarlyConnection, pconn model.UDPLikeConn) *quicConnectionOwnsConn {
	return &quicConnectionOwnsConn{EarlyConnection: qconn, conn: pconn}
}

// CloseWithError implements quic.EarlyConnection.CloseWithError.
func (qconn *quicConnectionOwnsConn) CloseWithError(
	code quic.ApplicationErrorCode, reason string) error {
	err := qconn.EarlyConnection.CloseWithError(code, reason)
	qconn.conn.Close()
	return err
}

// quicDialerResolver is a dialer that uses the configured Resolver
// to resolve a domain name to IP addrs.
type quicDialerResolver struct {
	// Dialer is the underlying QUICDialer.
	Dialer model.QUICDialer

	// Resolver is the underlying Resolver.
	Resolver model.Resolver
}

var _ model.QUICDialer = &quicDialerResolver{}

// DialContext implements QUICDialer.DialContext. This function
// will apply the following TLS defaults:
//
// 1. if tlsConfig.ServerName is empty, we will use the hostname
// contained inside of the `address` endpoint.
func (d *quicDialerResolver) DialContext(
	ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := d.lookupHost(ctx, onlyhost)
	if err != nil {
		return nil, err
	}
	tlsConfig = d.maybeApplyTLSDefaults(tlsConfig, onlyhost)
	// See TODO(https://github.com/ooni/probe/issues/1779) however
	// this is less of a problem for QUIC because so far we have been
	// using it to perform research only (i.e., urlgetter).
	addrs = quirkSortIPAddrs(addrs)
	var errorslist []error
	for _, addr := range addrs {
		target := net.JoinHostPort(addr, onlyport)
		qconn, err := d.Dialer.DialContext(
			ctx, network, target, tlsConfig, quicConfig)
		if err == nil {
			return qconn, nil
		}
		errorslist = append(errorslist, err)
	}
	return nil, quirkReduceErrors(errorslist)
}

// maybeApplyTLSDefaults sets the SNI if it's not already configured.
func (d *quicDialerResolver) maybeApplyTLSDefaults(config *tls.Config, host string) *tls.Config {
	config = config.Clone()
	if config.ServerName == "" {
		config.ServerName = host
	}
	return config
}

// lookupHost performs a domain name resolution.
func (d *quicDialerResolver) lookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return d.Resolver.LookupHost(ctx, hostname)
}

// CloseIdleConnections implements QUICDialer.CloseIdleConnections.
func (d *quicDialerResolver) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
	d.Resolver.CloseIdleConnections()
}

// quicDialerLogger is a dialer with logging.
type quicDialerLogger struct {
	// Dialer is the underlying QUIC dialer.
	Dialer model.QUICDialer

	// Logger is the underlying logger.
	Logger model.DebugLogger

	// operationSuffix is appended to the operation name.
	//
	// We use this suffix to distinguish the output from dialing
	// with the output from dialing an IP address when we are
	// using a dialer without resolver, where otherwise both lines
	// would read something like `dial 8.8.8.8:443...`
	operationSuffix string
}

var _ model.QUICDialer = &quicDialerLogger{}

// DialContext implements QUICContextDialer.DialContext.
func (d *quicDialerLogger) DialContext(
	ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
	d.Logger.Debugf("quic_dial%s %s/%s...", d.operationSuffix, address, network)
	qconn, err := d.Dialer.DialContext(ctx, network, address, tlsConfig, quicConfig)
	if err != nil {
		d.Logger.Debugf("quic_dial%s %s/%s... %s", d.operationSuffix,
			address, network, err)
		return nil, err
	}
	d.Logger.Debugf("quic_dial%s %s/%s... ok", d.operationSuffix, address, network)
	return qconn, nil
}

// CloseIdleConnections implements QUICDialer.CloseIdleConnections.
func (d *quicDialerLogger) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// NewSingleUseQUICDialer is like NewSingleUseDialer but for QUIC.
func NewSingleUseQUICDialer(qconn quic.EarlyConnection) model.QUICDialer {
	return &quicDialerSingleUse{qconn: qconn}
}

// quicDialerSingleUse is the QUICDialer returned by NewSingleQUICDialer.
type quicDialerSingleUse struct {
	mu    sync.Mutex
	qconn quic.EarlyConnection
}

var _ model.QUICDialer = &quicDialerSingleUse{}

// DialContext implements QUICDialer.DialContext.
func (s *quicDialerSingleUse) DialContext(
	ctx context.Context, network, addr string, tlsCfg *tls.Config,
	cfg *quic.Config) (quic.EarlyConnection, error) {
	var qconn quic.EarlyConnection
	defer s.mu.Unlock()
	s.mu.Lock()
	if s.qconn == nil {
		return nil, ErrNoConnReuse
	}
	qconn, s.qconn = s.qconn, nil
	return qconn, nil
}

// CloseIdleConnections closes idle connections.
func (s *quicDialerSingleUse) CloseIdleConnections() {
	// nothing to do
}

// quicListenerErrWrapper is a QUICListener that wraps errors.
type quicListenerErrWrapper struct {
	// QUICListener is the underlying listener.
	QUICListener model.QUICListener
}

var _ model.QUICListener = &quicListenerErrWrapper{}

// Listen implements QUICListener.Listen.
func (qls *quicListenerErrWrapper) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, NewErrWrapper(ClassifyGenericError, QUICListenOperation, err)
	}
	return &quicErrWrapperUDPLikeConn{pconn}, nil
}

// quicErrWrapperUDPLikeConn is a UDPLikeConn that wraps errors.
type quicErrWrapperUDPLikeConn struct {
	// UDPLikeConn is the underlying conn.
	model.UDPLikeConn
}

var _ model.UDPLikeConn = &quicErrWrapperUDPLikeConn{}

// WriteTo implements UDPLikeConn.WriteTo.
func (c *quicErrWrapperUDPLikeConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	if err != nil {
		return 0, NewErrWrapper(ClassifyGenericError, WriteToOperation, err)
	}
	return count, nil
}

// ReadFrom implements UDPLikeConn.ReadFrom.
func (c *quicErrWrapperUDPLikeConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.UDPLikeConn.ReadFrom(b)
	if err != nil {
		return 0, nil, NewErrWrapper(ClassifyGenericError, ReadFromOperation, err)
	}
	return n, addr, nil
}

// Close implements UDPLikeConn.Close.
func (c *quicErrWrapperUDPLikeConn) Close() error {
	err := c.UDPLikeConn.Close()
	if err != nil {
		return NewErrWrapper(ClassifyGenericError, ReadFromOperation, err)
	}
	return nil
}

// quicDialerErrWrapper is a dialer that performs quic err wrapping
type quicDialerErrWrapper struct {
	QUICDialer model.QUICDialer
}

var _ model.QUICDialer = &quicDialerErrWrapper{}

// DialContext implements ContextDialer.DialContext
func (d *quicDialerErrWrapper) DialContext(
	ctx context.Context, network string, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
	qconn, err := d.QUICDialer.DialContext(ctx, network, host, tlsCfg, cfg)
	if err != nil {
		return nil, NewErrWrapper(
			ClassifyQUICHandshakeError, QUICHandshakeOperation, err)
	}
	return qconn, nil
}

func (d *quicDialerErrWrapper) CloseIdleConnections() {
	d.QUICDialer.CloseIdleConnections()
}
