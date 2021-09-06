package netxlite

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// Dialer establishes network connections.
type Dialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}

// NewDialerWithResolver creates a dialer using the given resolver and logger.
func NewDialerWithResolver(logger Logger, resolver Resolver) Dialer {
	return &dialerLogger{
		Dialer: &dialerResolver{
			Dialer: &dialerLogger{
				Dialer:          &dialerSystem{},
				Logger:          logger,
				operationSuffix: "_address",
			},
			Resolver: resolver,
		},
		Logger: logger,
	}
}

// NewDialerWithoutResolver creates a dialer that uses the given
// logger and fails with ErrNoResolver when it is passed a domain name.
func NewDialerWithoutResolver(logger Logger) Dialer {
	return NewDialerWithResolver(logger, &nullResolver{})
}

// underlyingDialer is the Dialer we use by default.
var underlyingDialer = &net.Dialer{
	Timeout:   15 * time.Second,
	KeepAlive: 15 * time.Second,
}

// dialerSystem dials using Go stdlib.
type dialerSystem struct{}

// DialContext implements Dialer.DialContext.
func (d *dialerSystem) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return underlyingDialer.DialContext(ctx, network, address)
}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *dialerSystem) CloseIdleConnections() {
	// nothing
}

var defaultDialer Dialer = &dialerSystem{}

// dialerResolver is a dialer that uses the configured Resolver to resolver a
// domain name to IP addresses, and the configured Dialer to connect.
type dialerResolver struct {
	// Dialer is the underlying Dialer.
	Dialer Dialer

	// Resolver is the underlying Resolver.
	Resolver Resolver
}

var _ Dialer = &dialerResolver{}

// DialContext implements Dialer.DialContext.
func (d *dialerResolver) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := d.lookupHost(ctx, onlyhost)
	if err != nil {
		return nil, err
	}
	// TODO(bassosimone): here we should be using multierror rather
	// than just calling ReduceErrors. We are not ready to do that
	// yet, though. To do that, we need first to modify nettests so
	// that we actually avoid dialing when measuring.
	//
	// See also the quirks.go file. This is clearly a QUIRK.
	addrs = quirkSortIPAddrs(addrs)
	var errorslist []error
	for _, addr := range addrs {
		target := net.JoinHostPort(addr, onlyport)
		conn, err := d.Dialer.DialContext(ctx, network, target)
		if err == nil {
			return conn, nil
		}
		errorslist = append(errorslist, err)
	}
	return nil, quirkReduceErrors(errorslist)
}

// lookupHost performs a domain name resolution.
func (d *dialerResolver) lookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return d.Resolver.LookupHost(ctx, hostname)
}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *dialerResolver) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
	d.Resolver.CloseIdleConnections()
}

// dialerLogger is a Dialer with logging.
type dialerLogger struct {
	// Dialer is the underlying dialer.
	Dialer Dialer

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

var _ Dialer = &dialerLogger{}

// DialContext implements Dialer.DialContext
func (d *dialerLogger) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.Logger.Debugf("dial%s %s/%s...", d.operationSuffix, address, network)
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	elapsed := time.Since(start)
	if err != nil {
		d.Logger.Debugf("dial%s %s/%s... %s in %s", d.operationSuffix,
			address, network, err, elapsed)
		return nil, err
	}
	d.Logger.Debugf("dial%s %s/%s... ok in %s", d.operationSuffix,
		address, network, elapsed)
	return conn, nil
}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *dialerLogger) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// ErrNoConnReuse indicates we cannot reuse the connection provided
// to a single use (possibly TLS) dialer.
var ErrNoConnReuse = errors.New("cannot reuse connection")

// NewSingleUseDialer returns a dialer that returns the given connection once
// and after that always fails with the ErrNoConnReuse error.
func NewSingleUseDialer(conn net.Conn) Dialer {
	return &dialerSingleUse{conn: conn}
}

// dialerSingleUse is the type of Dialer returned by NewSingleDialer.
type dialerSingleUse struct {
	sync.Mutex
	conn net.Conn
}

var _ Dialer = &dialerSingleUse{}

// DialContext implements Dialer.DialContext.
func (s *dialerSingleUse) DialContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	defer s.Unlock()
	s.Lock()
	if s.conn == nil {
		return nil, ErrNoConnReuse
	}
	var conn net.Conn
	conn, s.conn = s.conn, nil
	return conn, nil
}

// CloseIdleConnections closes idle connections.
func (s *dialerSingleUse) CloseIdleConnections() {
	// nothing
}
