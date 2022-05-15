package netxlite

//
// Code for dialing TCP or UDP net.Conn-like connections
//

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewDialerWithResolver calls WrapDialer for the stdlib dialer.
func NewDialerWithResolver(logger model.DebugLogger, resolver model.Resolver) model.Dialer {
	return WrapDialer(logger, resolver, &dialerSystem{})
}

// WrapDialer creates a new Dialer that wraps the given
// Dialer. The returned Dialer has the following properties:
//
// 1. logs events using the given logger;
//
// 2. resolves domain names using the givern resolver;
//
// 3. when the resolver is not a "null" resolver,
// each available enpoint is tried
// sequentially. On error, the code will return what it believes
// to be the most representative error in the pack. Most often,
// the first error that occurred. Choosing the
// error to return using this logic is a QUIRK that we owe
// to the original implementation of netx. We cannot change
// this behavior until we refactor legacy code using it.
//
// Removing this quirk from the codebase is documented as
// TODO(https://github.com/ooni/probe/issues/1779).
//
// 4. wraps errors;
//
// 5. has a configured connect timeout;
//
// 6. if a dialer wraps a resolver, the dialer will forward
// the CloseIdleConnection call to its resolver (which is
// instrumental to manage a DoH resolver connections properly).
//
// In general, do not use WrapDialer directly but try to use
// more high-level factories, e.g., NewDialerWithResolver.
func WrapDialer(logger model.DebugLogger, resolver model.Resolver, dialer model.Dialer) model.Dialer {
	return &dialerLogger{
		Dialer: &dialerResolver{
			Dialer: &dialerLogger{
				Dialer: &dialerErrWrapper{
					Dialer: dialer,
				},
				DebugLogger:     logger,
				operationSuffix: "_address",
			},
			Resolver: resolver,
		},
		DebugLogger: logger,
	}
}

// NewDialerWithoutResolver calls NewDialerWithResolver with a "null" resolver.
//
// The returned dialer fails with ErrNoResolver if passed a domain name.
func NewDialerWithoutResolver(logger model.DebugLogger) model.Dialer {
	return NewDialerWithResolver(logger, &nullResolver{})
}

// dialerSystem uses system facilities to perform domain name
// resolution and guarantees we have a dialer timeout.
type dialerSystem struct {
	// timeout is the OPTIONAL timeout used for testing.
	timeout time.Duration
}

var _ model.Dialer = &dialerSystem{}

const dialerDefaultTimeout = 15 * time.Second

func (d *dialerSystem) newUnderlyingDialer() model.SimpleDialer {
	t := d.timeout
	if t <= 0 {
		t = dialerDefaultTimeout
	}
	return TProxy.NewSimpleDialer(t)
}

func (d *dialerSystem) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.newUnderlyingDialer().DialContext(ctx, network, address)
}

func (d *dialerSystem) CloseIdleConnections() {
	// nothing to do here
}

// dialerResolver combines dialing with domain name resolution.
type dialerResolver struct {
	Dialer   model.Dialer
	Resolver model.Resolver
}

var _ model.Dialer = &dialerResolver{}

func (d *dialerResolver) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	// QUIRK: this routine and the related routines in quirks.go cannot
	// be changed easily until we use events tracing to measure.
	//
	// Reference issue: TODO(https://github.com/ooni/probe/issues/1779).
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := d.lookupHost(ctx, onlyhost)
	if err != nil {
		return nil, err
	}
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

// lookupHost ensures we correctly handle IP addresses.
func (d *dialerResolver) lookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return d.Resolver.LookupHost(ctx, hostname)
}

func (d *dialerResolver) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
	d.Resolver.CloseIdleConnections()
}

// dialerLogger is a Dialer with logging.
type dialerLogger struct {
	// Dialer is the underlying dialer.
	Dialer model.Dialer

	// DebugLogger is the underlying logger.
	DebugLogger model.DebugLogger

	// operationSuffix is appended to the operation name.
	//
	// We use this suffix to distinguish the output from dialing
	// with the output from dialing an IP address when we are
	// using a dialer without resolver, where otherwise both lines
	// would read something like `dial 8.8.8.8:443...`
	operationSuffix string
}

var _ model.Dialer = &dialerLogger{}

func (d *dialerLogger) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.DebugLogger.Debugf("dial%s %s/%s...", d.operationSuffix, address, network)
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	elapsed := time.Since(start)
	if err != nil {
		d.DebugLogger.Debugf("dial%s %s/%s... %s in %s", d.operationSuffix,
			address, network, err, elapsed)
		return nil, err
	}
	d.DebugLogger.Debugf("dial%s %s/%s... ok in %s", d.operationSuffix,
		address, network, elapsed)
	return conn, nil
}

func (d *dialerLogger) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// ErrNoConnReuse is the type of error returned when you create a
// "single use" dialer or a "single use" TLS dialer and you dial
// more than once, which is not supported by such a dialer.
var ErrNoConnReuse = errors.New("cannot reuse connection")

// NewSingleUseDialer returns a "single use" dialer. The first
// dial will succed and return conn regardless of the network
// and address arguments passed to DialContext. Any subsequent
// dial returns ErrNoConnReuse.
func NewSingleUseDialer(conn net.Conn) model.Dialer {
	return &dialerSingleUse{conn: conn}
}

// dialerSingleUse is the Dialer returned by NewSingleDialer.
type dialerSingleUse struct {
	mu   sync.Mutex
	conn net.Conn
}

var _ model.Dialer = &dialerSingleUse{}

func (s *dialerSingleUse) DialContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	defer s.mu.Unlock()
	s.mu.Lock()
	if s.conn == nil {
		return nil, ErrNoConnReuse
	}
	var conn net.Conn
	conn, s.conn = s.conn, nil
	return conn, nil
}

func (s *dialerSingleUse) CloseIdleConnections() {
	// nothing to do
}

// dialerErrWrapper is a dialer that performs error wrapping. The connection
// returned by the DialContext function will also perform error wrapping.
type dialerErrWrapper struct {
	Dialer model.Dialer
}

var _ model.Dialer = &dialerErrWrapper{}

func (d *dialerErrWrapper) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, newErrWrapper(classifyGenericError, ConnectOperation, err)
	}
	return &dialerErrWrapperConn{Conn: conn}, nil
}

func (d *dialerErrWrapper) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// dialerErrWrapperConn is a net.Conn that performs error wrapping.
type dialerErrWrapperConn struct {
	net.Conn
}

var _ net.Conn = &dialerErrWrapperConn{}

func (c *dialerErrWrapperConn) Read(b []byte) (int, error) {
	count, err := c.Conn.Read(b)
	if err != nil {
		return 0, newErrWrapper(classifyGenericError, ReadOperation, err)
	}
	return count, nil
}

func (c *dialerErrWrapperConn) Write(b []byte) (int, error) {
	count, err := c.Conn.Write(b)
	if err != nil {
		return 0, newErrWrapper(classifyGenericError, WriteOperation, err)
	}
	return count, nil
}

func (c *dialerErrWrapperConn) Close() error {
	err := c.Conn.Close()
	if err != nil {
		return newErrWrapper(classifyGenericError, CloseOperation, err)
	}
	return nil
}

// ErrNoDialer is the type of error returned by "null" dialers
// when you attempt to dial with them.
var ErrNoDialer = errors.New("no configured dialer")

// NewNullDialer returns a dialer that always fails with ErrNoDialer.
func NewNullDialer() model.Dialer {
	return &nullDialer{}
}

type nullDialer struct{}

var _ model.Dialer = &nullDialer{}

func (*nullDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return nil, ErrNoDialer
}

func (*nullDialer) CloseIdleConnections() {
	// nothing to do
}
