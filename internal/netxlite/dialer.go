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

// DialerWrapper is a function that allows you to customize the kind of Dialer returned
// by WrapDialer, NewDialerWithResolver, and NewDialerWithoutResolver.
type DialerWrapper func(dialer model.Dialer) model.Dialer

// NewDialerWithResolver is equivalent to calling WrapDialer with
// the dialer argument being equal to &DialerSystem{}.
func NewDialerWithResolver(dl model.DebugLogger, r model.Resolver, w ...DialerWrapper) model.Dialer {
	return WrapDialer(dl, r, &DialerSystem{}, w...)
}

// WrapDialer wraps an existing Dialer to add extra functionality
// such as separting DNS lookup and connecting, error wrapping, logging, etc.
//
// When possible use NewDialerWithResolver or NewDialerWithoutResolver
// instead of using this rather low-level function.
//
// Arguments
//
// 1. logger is used to emit debug messages (MUST NOT be nil);
//
// 2. resolver is the resolver to use when dialing for endpoint
// addresses containing domain names (MUST NOT be nil);
//
// 3. baseDialer is the dialer to wrap (MUST NOT be nil);
//
// 4. wrappers is a list of zero or more functions allowing you to
// modify the behavior of the returned dialer (see below).
//
// Return value
//
// The returned dialer is an opaque type consisting of the composition of
// several simple dialers. The following pseudo code illustrates the general
// behavior of the returned composed dialer:
//
//     addrs, err := dnslookup()
//     if err != nil {
//       return nil, err
//     }
//     errors := []error{}
//     for _, a := range addrs {
//       conn, err := tcpconnect(a)
//       if err != nil {
//         errors = append(errors, err)
//         continue
//       }
//       return conn, nil
//     }
//     return nil, errors[0]
//
//
// The following table describes the structure of the returned dialer:
//
//     +-------+-----------------+------------------------------------------+
//     | Index | Name            | Description                              |
//     +-------+-----------------+------------------------------------------+
//     | 0     | base            | the baseDialer argument                  |
//     +-------+-----------------+------------------------------------------+
//     | 1     | errWrapper      | wraps Go errors to be consistent with    |
//     |       |                 | OONI df-007-errors spec                  |
//     +-------+-----------------+------------------------------------------+
//     | 2     | ???             | if there are wrappers, result of calling |
//     |       |                 | the first one on the errWrapper dialer   |
//     +-------+-----------------+------------------------------------------+
//     | ...   | ...             | ...                                      |
//     +-------+-----------------+------------------------------------------+
//     | N     | ???             | if there are wrappers, result of calling |
//     |       |                 | the last one on the N-1 dialer           |
//     +-------+-----------------+------------------------------------------+
//     | N+1   | logger (inner)  | logs TCP connect operations              |
//     +-------+-----------------+------------------------------------------+
//     | N+2   | resolver        | DNS lookup and try connect each IP in    |
//     |       |                 | sequence until one of them succeeds      |
//     +-------+-----------------+------------------------------------------+
//     | N+3   | logger (outer)  | logs the overall dial operation          |
//     +-------+-----------------+------------------------------------------+
//
// The list of wrappers allows to insert modified dialers in the correct
// place for observing and saving I/O events (connect, read, etc.).
//
// Remarks
//
// When the resolver is &NullResolver{} any attempt to perform DNS resolutions
// in the dialer at index N+2 will fail with ErrNoResolver.
//
// Otherwise, the dialer at index N+2 will try each resolver IP address
// sequentially. In case of failure, such a resolver will return the first
// error that occurred. This implementation strategy is a QUIRK that is
// documented at TODO(https://github.com/ooni/probe/issues/1779).
//
// If the baseDialer is &DialerSystem{}, there will be a fixed TCP connect
// timeout for each connect operation. Because there may be multiple IP
// addresses per dial, the overall timeout would be a multiple of the timeout
// of a single connect operation. You may want to use the context to reduce
// the overall time spent trying all addresses and timing out.
func WrapDialer(logger model.DebugLogger, resolver model.Resolver,
	baseDialer model.Dialer, wrappers ...DialerWrapper) (outDialer model.Dialer) {
	outDialer = &dialerErrWrapper{
		Dialer: baseDialer,
	}
	for _, wrapper := range wrappers {
		outDialer = wrapper(outDialer) // extend with user-supplied constructors
	}
	return &dialerLogger{
		Dialer: &dialerResolver{
			Dialer: &dialerLogger{
				Dialer:          outDialer,
				DebugLogger:     logger,
				operationSuffix: "_address",
			},
			Resolver: resolver,
		},
		DebugLogger: logger,
	}
}

// NewDialerWithoutResolver is equivalent to calling NewDialerWithResolver
// with the resolver argument being &NullResolver{}.
func NewDialerWithoutResolver(dl model.DebugLogger, w ...DialerWrapper) model.Dialer {
	return NewDialerWithResolver(dl, &NullResolver{}, w...)
}

// DialerSystem is a model.Dialer that users TProxy.NewSimplerDialer
// to construct the new SimpleDialer used for dialing. This dialer has
// a fixed timeout for each connect operation equal to 15 seconds.
type DialerSystem struct {
	// timeout is the OPTIONAL timeout (for testing).
	timeout time.Duration
}

var _ model.Dialer = &DialerSystem{}

const dialerDefaultTimeout = 15 * time.Second

func (d *DialerSystem) newUnderlyingDialer() model.SimpleDialer {
	t := d.timeout
	if t <= 0 {
		t = dialerDefaultTimeout
	}
	return TProxy.NewSimpleDialer(t)
}

func (d *DialerSystem) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.newUnderlyingDialer().DialContext(ctx, network, address)
}

func (d *DialerSystem) CloseIdleConnections() {
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
