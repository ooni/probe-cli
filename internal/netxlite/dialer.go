package netxlite

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// Dialer establishes network connections.
type Dialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}

// NewDialerWithResolver is a convenience factory that calls
// WrapDialer for a stdlib dialer type.
func NewDialerWithResolver(logger Logger, resolver Resolver) Dialer {
	return WrapDialer(logger, resolver, &dialerSystem{})
}

// WrapDialer creates a new Dialer that wraps the given
// Dialer. The returned Dialer has the following properties:
//
// 1. logs events using the given logger;
//
// 2. resolves domain names using the givern resolver;
//
// 3. when using a resolver, each available enpoint is tried
// sequentially. On error, the code will return what it believes
// to be the most representative error in the pack. Most often,
// such an error is the first one that occurred. Choosing the
// error to return using this logic is a QUIRK that we owe
// to the original implementation of netx. We cannot change
// this behavior until all the legacy code that relies on
// it has been migrated to more sane patterns.
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
func WrapDialer(logger Logger, resolver Resolver, dialer Dialer) Dialer {
	return &dialerLogger{
		Dialer: &dialerResolver{
			Dialer: &dialerLogger{
				Dialer: &dialerErrWrapper{
					Dialer: dialer,
				},
				Logger:          logger,
				operationSuffix: "_address",
			},
			Resolver: resolver,
		},
		Logger: logger,
	}
}

// NewDialerWithoutResolver is like NewDialerWithResolver except that
// it will fail with ErrNoResolver if passed a domain name.
func NewDialerWithoutResolver(logger Logger) Dialer {
	return NewDialerWithResolver(logger, &nullResolver{})
}

// dialerSystem uses system facilities to perform domain name
// resolution and guarantees we have a dialer timeout.
type dialerSystem struct {
	// timeout is the OPTIONAL timeout used for testing.
	timeout time.Duration
}

var _ Dialer = &dialerSystem{}

const dialerDefaultTimeout = 15 * time.Second

func (d *dialerSystem) newUnderlyingDialer() *net.Dialer {
	t := d.timeout
	if t <= 0 {
		t = dialerDefaultTimeout
	}
	return &net.Dialer{Timeout: t}
}

func (d *dialerSystem) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.newUnderlyingDialer().DialContext(ctx, network, address)
}

func (d *dialerSystem) CloseIdleConnections() {
	// nothing to do here
}

// dialerResolver combines dialing with domain name resolution.
type dialerResolver struct {
	Dialer
	Resolver
}

var _ Dialer = &dialerResolver{}

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
	Dialer

	// Logger is the underlying logger.
	Logger

	// operationSuffix is appended to the operation name.
	//
	// We use this suffix to distinguish the output from dialing
	// with the output from dialing an IP address when we are
	// using a dialer without resolver, where otherwise both lines
	// would read something like `dial 8.8.8.8:443...`
	operationSuffix string
}

var _ Dialer = &dialerLogger{}

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

// dialerSingleUse is the Dialer returned by NewSingleDialer.
type dialerSingleUse struct {
	sync.Mutex
	conn net.Conn
}

var _ Dialer = &dialerSingleUse{}

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

func (s *dialerSingleUse) CloseIdleConnections() {
	// nothing to do
}

// dialerErrWrapper is a dialer that performs error wrapping. The connection
// returned by the DialContext function will also perform error wrapping.
type dialerErrWrapper struct {
	Dialer
}

var _ Dialer = &dialerErrWrapper{}

func (d *dialerErrWrapper) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError, errorsx.ConnectOperation, err)
	}
	return &dialerErrWrapperConn{Conn: conn}, nil
}

// dialerErrWrapperConn is a net.Conn that performs error wrapping.
type dialerErrWrapperConn struct {
	net.Conn
}

var _ net.Conn = &dialerErrWrapperConn{}

func (c *dialerErrWrapperConn) Read(b []byte) (int, error) {
	count, err := c.Conn.Read(b)
	if err != nil {
		return 0, errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError, errorsx.ReadOperation, err)
	}
	return count, nil
}

func (c *dialerErrWrapperConn) Write(b []byte) (int, error) {
	count, err := c.Conn.Write(b)
	if err != nil {
		return 0, errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError, errorsx.WriteOperation, err)
	}
	return count, nil
}

func (c *dialerErrWrapperConn) Close() error {
	err := c.Conn.Close()
	if err != nil {
		return errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError, errorsx.CloseOperation, err)
	}
	return nil
}

// ErrNoDialer indicates that no dialer is configured.
var ErrNoDialer = errors.New("no configured dialer")

// NewNullDialer returns a dialer that always fails.
func NewNullDialer() Dialer {
	return &nullDialer{}
}

type nullDialer struct{}

var _ Dialer = &nullDialer{}

func (*nullDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return nil, ErrNoDialer
}

func (*nullDialer) CloseIdleConnections() {
	// nothing to do
}
