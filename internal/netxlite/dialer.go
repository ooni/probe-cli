package netxlite

import (
	"context"
	"net"
	"time"
)

// Dialer establishes network connections.
type Dialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
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
	var errorslist []error
	for _, addr := range addrs {
		target := net.JoinHostPort(addr, onlyport)
		conn, err := d.Dialer.DialContext(ctx, network, target)
		if err == nil {
			return conn, nil
		}
		errorslist = append(errorslist, err)
	}
	return nil, reduceErrors(errorslist)
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
}

var _ Dialer = &dialerLogger{}

// DialContext implements Dialer.DialContext
func (d *dialerLogger) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.Logger.Debugf("dial %s/%s...", address, network)
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	elapsed := time.Since(start)
	if err != nil {
		d.Logger.Debugf("dial %s/%s... %s in %s", address, network, err, elapsed)
		return nil, err
	}
	d.Logger.Debugf("dial %s/%s... ok in %s", address, network, elapsed)
	return conn, nil
}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *dialerLogger) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}
