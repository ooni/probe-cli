package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// QUICContextDialer is a dialer for QUIC using Context.
type QUICContextDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	DialContext(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)
}

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening UDPConn.
	Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error)
}

// quicListenerStdlib is a QUICListener using the standard library.
type quicListenerStdlib struct{}

var _ QUICListener = &quicListenerStdlib{}

// Listen implements QUICListener.Listen.
func (qls *quicListenerStdlib) Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	return net.ListenUDP("udp", addr)
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

var _ QUICContextDialer = &quicDialerQUICGo{}

// errInvalidIP indicates that a string is not a valid IP.
var errInvalidIP = errors.New("netxlite: invalid IP")

// DialContext implements ContextDialer.DialContext. This function will
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
	// Dialer is the underlying QUIC dialer.
	Dialer QUICContextDialer

	// Resolver is the underlying resolver.
	Resolver Resolver
}

var _ QUICContextDialer = &quicDialerResolver{}

// DialContext implements QUICContextDialer.DialContext. This function
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
	// TODO(bassosimone): here we should be using multierror rather
	// than just calling ReduceErrors. We are not ready to do that
	// yet, though. To do that, we need first to modify nettests so
	// that we actually avoid dialing when measuring.
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
	return nil, reduceErrors(errorslist)
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

// quicDialerLogger is a dialer with logging.
type quicDialerLogger struct {
	// Dialer is the underlying QUIC dialer.
	Dialer QUICContextDialer

	// Logger is the underlying logger.
	Logger Logger
}

var _ QUICContextDialer = &quicDialerLogger{}

// DialContext implements QUICContextDialer.DialContext.
func (d *quicDialerLogger) DialContext(
	ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	d.Logger.Debugf("quic %s/%s...", address, network)
	sess, err := d.Dialer.DialContext(ctx, network, address, tlsConfig, quicConfig)
	if err != nil {
		d.Logger.Debugf("quic %s/%s... %s", address, network, err)
		return nil, err
	}
	d.Logger.Debugf("quic %s/%s... ok", address, network)
	return sess, nil
}
