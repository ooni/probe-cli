package netplumbing

import (
	"context"
	"net"
	"time"
)

// Connector creates new network connections.
type Connector interface {
	// DialContext establishes a new network connection.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// DefaultConnector is the connector used by default.
var DefaultConnector = &net.Dialer{
	// Timeout is the connect timeout.
	Timeout: 15 * time.Second,

	// KeepAlive is the keep-alive interval (may not work on all
	// platforms, as it depends on kernel support).
	KeepAlive: 30 * time.Second,
}

// ErrConnect is a connect error.
type ErrConnect struct {
	error
}

// Unwrap returns the underlying error.
func (e *ErrConnect) Unwrap() error {
	return e.error
}

// connect establishes a new network connection.
func (txp *Transport) connect(ctx context.Context, network, address string) (net.Conn, error) {
	fn := DefaultConnector.DialContext
	if settings := ContextSettings(ctx); settings != nil && settings.Connector != nil {
		fn = settings.Connector.DialContext
	}
	conn, err := fn(ctx, network, address)
	if err != nil {
		return nil, &ErrConnect{err}
	}
	return &connWrapper{byteCounter: txp.byteCounter(ctx), Conn: conn}, nil
}

// noopByteCounter is a no-op ByteCounter.
type noopByteCounter struct{}

// CountyBytesReceived increments the bytes received count.
func (*noopByteCounter) CountBytesReceived(count int) {}

// CountBytesSent increments the bytes sent count.
func (*noopByteCounter) CountBytesSent(count int) {}

// defaultByteCounter is the default byte counter.
var defaultByteCounter = &noopByteCounter{}

// byteCounter returns the ByteCounter to use.
func (txp *Transport) byteCounter(ctx context.Context) ByteCounter {
	if settings := ContextSettings(ctx); settings != nil && settings.ByteCounter != nil {
		return settings.ByteCounter
	}
	return defaultByteCounter
}
