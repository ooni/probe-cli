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
	log := txp.logger(ctx)
	log.Debugf("connect: %s/%s...", address, network)
	fn := txp.DefaultConnector().DialContext
	if config := ContextConfig(ctx); config != nil && config.Connector != nil {
		fn = config.Connector.DialContext
	}
	conn, err := fn(ctx, network, address)
	if err != nil {
		log.Debugf("connect: %s/%s... %s", address, network, err)
		return nil, &ErrConnect{err}
	}
	log.Debugf("connect: %s/%s... ok", address, network)
	return &connWrapper{byteCounter: txp.byteCounter(ctx), Conn: conn}, nil
}

// noopByteCounter is a no-op ByteCounter.
type noopByteCounter struct{}

// CountyBytesReceived increments the bytes received count.
func (*noopByteCounter) CountBytesReceived(count int) {}

// CountBytesSent increments the bytes sent count.
func (*noopByteCounter) CountBytesSent(count int) {}

// byteCounter returns the ByteCounter to use.
func (txp *Transport) byteCounter(ctx context.Context) ByteCounter {
	if config := ContextConfig(ctx); config != nil && config.ByteCounter != nil {
		return config.ByteCounter
	}
	return &noopByteCounter{}
}
