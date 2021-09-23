package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"sync"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening UDPConn.
	Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error)
}

// NewQUICListener creates a new QUICListener using the standard
// library to create listening UDP sockets.
func NewQUICListener() QUICListener {
	return &quicListenerErrWrapper{&quicListenerStdlib{}}
}

// quicListenerStdlib is a QUICListener using the standard library.
type quicListenerStdlib struct{}

var _ QUICListener = &quicListenerStdlib{}

// Listen implements QUICListener.Listen.
func (qls *quicListenerStdlib) Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	return bwmonitor.MaybeWrapUDPLikeConn(net.ListenUDP("udp", addr))
}

// QUICDialer dials QUIC sessions.
type QUICDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	DialContext(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}

// NewQUICDialerWithResolver returns a QUICDialer using the given
// QUICListener to create listening connections and the given Resolver
// to resolve domain names (if needed).
//
// Properties of the dialer:
//
// 1. logs events using the given logger;
//
// 2. resolves domain names using the givern resolver;
//
// 3. when using a resolver, _may_ attempt multiple dials
// in parallel (happy eyeballs) and _may_ return an aggregate
// error to the caller;
//
// 4. wraps errors;
//
// 5. has a configured connect timeout;
//
// 6. if a dialer wraps a resolver, the dialer will forward
// the CloseIdleConnection call to its resolver (which is
// instrumental to manage a DoH resolver connections properly);
//
// 7. will use the bundled CA unless you provide another CA;
//
// 8. will attempt to guess SNI when resolving domain names
// and otherwise will not set the SNI;
//
// 9. will attempt to guess ALPN when the port is known and
// otherwise will not set the ALPN.
func NewQUICDialerWithResolver(listener QUICListener,
	logger Logger, resolver Resolver) QUICDialer {
	return &quicDialerLogger{
		Dialer: &quicDialerResolver{
			Dialer: &quicDialerLogger{
				Dialer: &quicDialerErrWrapper{
					QUICDialer: &quicDialerQUICGo{
						QUICListener: listener,
					}},
				Logger:          logger,
				operationSuffix: "_address",
			},
			Resolver: resolver,
		},
		Logger: logger,
	}
}

// NewQUICDialerWithoutResolver is like NewQUICDialerWithResolver
// except that there is no configured resolver. So, if you pass in
// an address containing a domain name, the dial will fail with
// the ErrNoResolver failure.
func NewQUICDialerWithoutResolver(listener QUICListener, logger Logger) QUICDialer {
	return NewQUICDialerWithResolver(listener, logger, &nullResolver{})
}

// quicDialerQUICGo dials using the lucas-clemente/quic-go library.
type quicDialerQUICGo struct {
	// QUICListener is the underlying QUICListener to use.
	QUICListener QUICListener

	// mockDialEarlyContext allows to mock quic.DialEarlyContext.
	mockDialEarlyContext func(ctx context.Context, pconn net.PacketConn,
		remoteAddr net.Addr, host string, tlsConfig *tls.Config,
		quicConfig *quic.Config) (quic.EarlySession, error)
}

var _ QUICDialer = &quicDialerQUICGo{}

// errInvalidIP indicates that a string is not a valid IP.
var errInvalidIP = errors.New("netxlite: invalid IP")

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
	quic.EarlySession, error) {
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(onlyport)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(onlyhost)
	if ip == nil {
		return nil, errInvalidIP
	}
	pconn, err := d.QUICListener.Listen(&net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	udpAddr := &net.UDPAddr{IP: ip, Port: port, Zone: ""}
	tlsConfig = d.maybeApplyTLSDefaults(tlsConfig, port)
	sess, err := d.dialEarlyContext(
		ctx, pconn, udpAddr, address, tlsConfig, quicConfig)
	if err != nil {
		return nil, err
	}
	return &quicSessionOwnsConn{EarlySession: sess, conn: pconn}, nil
}

func (d *quicDialerQUICGo) dialEarlyContext(ctx context.Context,
	pconn net.PacketConn, remoteAddr net.Addr, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
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

// quicSessionOwnsConn ensures that we close the UDPLikeConn.
type quicSessionOwnsConn struct {
	// EarlySession is the embedded early session
	quic.EarlySession

	// conn is the connection we own
	conn quicx.UDPLikeConn
}

// CloseWithError implements quic.EarlySession.CloseWithError.
func (sess *quicSessionOwnsConn) CloseWithError(
	code quic.ApplicationErrorCode, reason string) error {
	err := sess.EarlySession.CloseWithError(code, reason)
	sess.conn.Close()
	return err
}

// quicDialerResolver is a dialer that uses the configured Resolver
// to resolve a domain name to IP addrs.
type quicDialerResolver struct {
	// Dialer is the underlying QUICDialer.
	Dialer QUICDialer

	// Resolver is the underlying Resolver.
	Resolver Resolver
}

var _ QUICDialer = &quicDialerResolver{}

// DialContext implements QUICDialer.DialContext. This function
// will apply the following TLS defaults:
//
// 1. if tlsConfig.ServerName is empty, we will use the hostname
// contained inside of the `address` endpoint.
func (d *quicDialerResolver) DialContext(
	ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
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
		sess, err := d.Dialer.DialContext(
			ctx, network, target, tlsConfig, quicConfig)
		if err == nil {
			return sess, nil
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
	Dialer QUICDialer

	// Logger is the underlying logger.
	Logger Logger

	// operationSuffix is appended to the operation name.
	//
	// We use this suffix to distinguish the output from dialing
	// with the output from dialing an IP address when we are
	// using a dialer without resolver, where otherwise both lines
	// would read something like `dial 8.8.8.8:443...`
	operationSuffix string
}

var _ QUICDialer = &quicDialerLogger{}

// DialContext implements QUICContextDialer.DialContext.
func (d *quicDialerLogger) DialContext(
	ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	d.Logger.Debugf("quic_dial%s %s/%s...", d.operationSuffix, address, network)
	sess, err := d.Dialer.DialContext(ctx, network, address, tlsConfig, quicConfig)
	if err != nil {
		d.Logger.Debugf("quic_dial%s %s/%s... %s", d.operationSuffix,
			address, network, err)
		return nil, err
	}
	d.Logger.Debugf("quic_dial%s %s/%s... ok", d.operationSuffix, address, network)
	return sess, nil
}

// CloseIdleConnections implements QUICDialer.CloseIdleConnections.
func (d *quicDialerLogger) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// NewSingleUseQUICDialer returns a dialer that returns the given connection
// once and after that always fails with the ErrNoConnReuse error.
func NewSingleUseQUICDialer(sess quic.EarlySession) QUICDialer {
	return &quicDialerSingleUse{sess: sess}
}

// quicDialerSingleUse is the QUICDialer returned by NewSingleQUICDialer.
type quicDialerSingleUse struct {
	sync.Mutex
	sess quic.EarlySession
}

var _ QUICDialer = &quicDialerSingleUse{}

// DialContext implements QUICDialer.DialContext.
func (s *quicDialerSingleUse) DialContext(
	ctx context.Context, network, addr string, tlsCfg *tls.Config,
	cfg *quic.Config) (quic.EarlySession, error) {
	var sess quic.EarlySession
	defer s.Unlock()
	s.Lock()
	if s.sess == nil {
		return nil, ErrNoConnReuse
	}
	sess, s.sess = s.sess, nil
	return sess, nil
}

// CloseIdleConnections closes idle connections.
func (s *quicDialerSingleUse) CloseIdleConnections() {
	// nothing to do
}

// quicListenerErrWrapper is a QUICListener that wraps errors.
type quicListenerErrWrapper struct {
	// QUICListener is the underlying listener.
	QUICListener
}

var _ QUICListener = &quicListenerErrWrapper{}

// Listen implements QUICListener.Listen.
func (qls *quicListenerErrWrapper) Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError, errorsx.QUICListenOperation, err)
	}
	return &quicErrWrapperUDPLikeConn{pconn}, nil
}

// quicErrWrapperUDPLikeConn is a quicx.UDPLikeConn that wraps errors.
type quicErrWrapperUDPLikeConn struct {
	// UDPLikeConn is the underlying conn.
	quicx.UDPLikeConn
}

var _ quicx.UDPLikeConn = &quicErrWrapperUDPLikeConn{}

// WriteTo implements quicx.UDPLikeConn.WriteTo.
func (c *quicErrWrapperUDPLikeConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	if err != nil {
		return 0, errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError, errorsx.WriteToOperation, err)
	}
	return count, nil
}

// ReadFrom implements quicx.UDPLikeConn.ReadFrom.
func (c *quicErrWrapperUDPLikeConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.UDPLikeConn.ReadFrom(b)
	if err != nil {
		return 0, nil, errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError, errorsx.ReadFromOperation, err)
	}
	return n, addr, nil
}

// Close implements quicx.UDPLikeConn.Close.
func (c *quicErrWrapperUDPLikeConn) Close() error {
	err := c.UDPLikeConn.Close()
	if err != nil {
		return errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError, errorsx.ReadFromOperation, err)
	}
	return nil
}

// quicDialerErrWrapper is a dialer that performs quic err wrapping
type quicDialerErrWrapper struct {
	QUICDialer
}

// DialContext implements ContextDialer.DialContext
func (d *quicDialerErrWrapper) DialContext(
	ctx context.Context, network string, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	sess, err := d.QUICDialer.DialContext(ctx, network, host, tlsCfg, cfg)
	if err != nil {
		return nil, errorsx.NewErrWrapper(
			errorsx.ClassifyQUICHandshakeError, errorsx.QUICHandshakeOperation, err)
	}
	return sess, nil
}
