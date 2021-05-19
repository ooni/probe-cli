package netplumbing

import (
	"context"
	"fmt"
	"net"
	"time"
)

// DialContext dials a cleartext connection.
func (txp *Transport) DialContext(
	ctx context.Context, network string, address string) (net.Conn, error) {
	return txp.dialContextWrapError(ctx, network, address)
}

// dialContextWrapError wraps any error using ErrDial.
func (txp *Transport) dialContextWrapError(
	ctx context.Context, network string, address string) (net.Conn, error) {
	conn, err := txp.dialContextMaybeProxy(ctx, network, address)
	if err != nil {
		return nil, &ErrDial{err}
	}
	return conn, nil
}

// ErrDial is an error occurred when dialing.
type ErrDial struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrDial) Unwrap() error {
	return err.error
}

// dialContextMaybeProxy chooses whether to use a proxy. We do not use
// any proxy when called by HTTP, because HTTP manages the proxy for itself.
func (txp *Transport) dialContextMaybeProxy(
	ctx context.Context, network string, address string) (net.Conn, error) {
	if config := ContextConfig(ctx); config != nil && config.Proxy != nil {
		return txp.dialProxy(ctx, network, address, config.Proxy)
	}
	return txp.dialContextEmitLogs(ctx, network, address)
}

// dialContextEmitLogs emits dial-related logs.
func (txp *Transport) dialContextEmitLogs(
	ctx context.Context, network string, address string) (net.Conn, error) {
	log := txp.logger(ctx)
	log.Debugf("dial: %s/%s...", address, network)
	conn, err := txp.dialContextResolveAndLoop(ctx, network, address)
	if err != nil {
		log.Debugf("dial: %s/%s... %s", address, network, err)
		return nil, err
	}
	log.Debugf("dial: %s/%s... ok", address, network)
	return conn, nil
}

// dialContextResolveAndLoop resolves the domain name in address
// to IP addresses, and tries every address until one of them
// succeeds or all of them have failed.
func (txp *Transport) dialContextResolveAndLoop(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	hostname, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ipaddrs, err := txp.LookupHost(ctx, hostname)
	if err != nil {
		return nil, err
	}
	aggregate := &ErrAllConnectsFailed{}
	for _, ipaddr := range ipaddrs {
		endpoint := net.JoinHostPort(ipaddr, port)
		conn, err := txp.connect(ctx, network, endpoint)
		if err == nil {
			return conn, nil
		}
		aggregate.Errors = append(aggregate.Errors, err)
	}
	return nil, aggregate
}

// ErrAllConnectsFailed indicates that all connects failed.
type ErrAllConnectsFailed struct {
	// Errors contains all the errors that occurred.
	Errors []error
}

// Error implements error.Error.
func (err *ErrAllConnectsFailed) Error() string {
	return fmt.Sprintf("one or more connect() failed: %#v", err.Errors)
}

// connect is the top-level operation for connecting to a TCP endpoint.
func (txp *Transport) connect(
	ctx context.Context, network, address string) (net.Conn, error) {
	return txp.connectWrapError(ctx, network, address)
}

// connectWrapError wraps eny error using ErrConnect.
func (txp *Transport) connectWrapError(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := txp.connectEmitLogs(ctx, network, address)
	if err != nil {
		return nil, &ErrConnect{err}
	}
	return conn, nil
}

// ErrConnect is a connect error.
type ErrConnect struct {
	error
}

// Unwrap returns the underlying error.
func (e *ErrConnect) Unwrap() error {
	return e.error
}

// connectEmitLogs emits logs related to connect.
func (txp *Transport) connectEmitLogs(
	ctx context.Context, network, address string) (net.Conn, error) {
	log := txp.logger(ctx)
	log.Debugf("connect: %s/%s...", address, network)
	conn, err := txp.connectWrapConn(ctx, network, address)
	if err != nil {
		log.Debugf("connect: %s/%s... %s", address, network, err)
		return nil, err
	}
	log.Debugf("connect: %s/%s... ok", address, network)
	return conn, nil
}

// connectWrapConn wraps the returned connection with connWrapper.
func (txp *Transport) connectWrapConn(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := txp.connectMaybeTrace(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &connWrapper{byteCounter: txp.byteCounter(ctx), Conn: conn}, nil
}

// connWrapper wraps all connections dialed using connect.
type connWrapper struct {
	byteCounter ByteCounter
	net.Conn
}

// Read implements net.Conn.Read. When this function returns an
// error it's always an ErrRead error.
func (conn *connWrapper) Read(b []byte) (int, error) {
	count, err := conn.Conn.Read(b)
	if err != nil {
		return 0, &ErrRead{err}
	}
	conn.byteCounter.CountBytesReceived(count)
	return count, nil
}

// ErrRead is a read error.
type ErrRead struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrRead) Unwrap() error {
	return err.error
}

// Write implements net.Conn.Write. When this function returns an
// error, it's always an ErrWrite error.
func (conn *connWrapper) Write(b []byte) (int, error) {
	count, err := conn.Conn.Write(b)
	if err != nil {
		return 0, &ErrWrite{err}
	}
	conn.byteCounter.CountBytesSent(count)
	return count, nil
}

// ErrWrite is a write error.
type ErrWrite struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrWrite) Unwrap() error {
	return err.error
}

// connectMaybeTrace enables tracing if needed.
func (txp *Transport) connectMaybeTrace(
	ctx context.Context, network, address string) (net.Conn, error) {
	if th := ContextTraceHeader(ctx); th != nil {
		return txp.connectWithTraceHeader(ctx, network, address, th)
	}
	return txp.connectMaybeOverride(ctx, network, address)
}

// connectWithTraceHeader traces a connect operation.
func (txp *Transport) connectWithTraceHeader(
	ctx context.Context, network, address string,
	th *TraceHeader) (net.Conn, error) {
	ev := &ConnectTrace{
		Network:    network,
		RemoteAddr: address,
		StartTime:  time.Now(),
	}
	defer th.add(ev)
	conn, err := txp.connectMaybeOverride(ctx, network, address)
	ev.EndTime = time.Now()
	if err != nil {
		ev.Error = err
		return nil, err
	}
	ev.LocalAddr = conn.LocalAddr().String()
	return &tracerConn{Conn: conn, th: th}, nil
}

// ConnectTrace is a measurement performed during connect.
type ConnectTrace struct {
	// Network is the network we're using (e.g., "tcp")
	Network string

	// RemoteAddr is the address we're connecting to.
	RemoteAddr string

	// StartTime is when we started connecting.
	StartTime time.Time

	// EndTime is when we're done.
	EndTime time.Time

	// LocalAddr is the local address in case of success.
	LocalAddr string

	// Error is the error that occurred.
	Error error
}

// Kind implements TraceEvent.Kind.
func (te *ConnectTrace) Kind() string {
	return TraceKindConnect
}

// tracerConn wraps a net.Conn when we need tracing.
type tracerConn struct {
	net.Conn
	th *TraceHeader
}

// ReadWriteTrace is a trace collected when reading or writing.
type ReadWriteTrace struct {
	// kind is the structure kind.
	kind string

	// LocalAddr is the local address.
	LocalAddr string

	// RemoteAddr is the remote address.
	RemoteAddr string

	// BufferSize is the size of the buffer to send or recv.
	BufferSize int

	// StartTime is when we started the resolve.
	StartTime time.Time

	// EndTime is when we're done. The duration of the round trip
	// also includes the time spent reading the response.
	EndTime time.Time

	// Count is the number of bytes read or written.
	Count int

	// Error is the error that occurred.
	Error error
}

// Kind implements TraceEvent.Kind.
func (te *ReadWriteTrace) Kind() string {
	return te.kind
}

// Read implements net.Conn.Read.
func (c *tracerConn) Read(b []byte) (int, error) {
	ev := &ReadWriteTrace{
		kind:       TraceKindRead,
		RemoteAddr: c.Conn.RemoteAddr().String(),
		LocalAddr:  c.Conn.LocalAddr().String(),
		BufferSize: len(b),
		StartTime:  time.Now(),
	}
	defer c.th.add(ev)
	count, err := c.Conn.Read(b)
	ev.EndTime = time.Now()
	ev.Count = count
	ev.Error = err
	return count, err
}

// Write implements net.Conn.Write.
func (c *tracerConn) Write(b []byte) (int, error) {
	ev := &ReadWriteTrace{
		kind:       TraceKindWrite,
		RemoteAddr: c.Conn.RemoteAddr().String(),
		LocalAddr:  c.Conn.LocalAddr().String(),
		BufferSize: len(b),
		StartTime:  time.Now(),
	}
	defer c.th.add(ev)
	count, err := c.Conn.Write(b)
	ev.EndTime = time.Now()
	ev.Count = count
	ev.Error = err
	return count, err
}

// connectMaybeOverride uses the default or the overriden connector.
func (txp *Transport) connectMaybeOverride(
	ctx context.Context, network, address string) (net.Conn, error) {
	fn := txp.DefaultConnector().DialContext
	if config := ContextConfig(ctx); config != nil && config.Connector != nil {
		fn = config.Connector.DialContext
	}
	return fn(ctx, network, address)
}

// DefaultConnector returns the default connector used by a transport.
func (txp *Transport) DefaultConnector() Connector {
	return &net.Dialer{
		// Timeout is the connect timeout.
		Timeout: 15 * time.Second,

		// KeepAlive is the keep-alive interval (may not work on all
		// platforms, as it depends on kernel support).
		KeepAlive: 30 * time.Second,
	}
}
