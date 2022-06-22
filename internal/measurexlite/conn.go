package measurexlite

//
// Conn tracing
//

import (
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// MaybeClose is a convenience function for closing a conn only when such a conn isn't nil.
func MaybeClose(conn net.Conn) (err error) {
	if conn != nil {
		err = conn.Close()
	}
	return
}

// WrapNetConn returns a wrapped conn that saves network events into this trace.
func (tx *Trace) WrapNetConn(conn net.Conn) net.Conn {
	return &connTrace{
		Conn: conn,
		tx:   tx,
	}
}

// connTrace is a trace-aware net.Conn.
type connTrace struct {
	// Implementation note: it seems safe to use embedding here because net.Conn
	// is an interface from the standard library that we don't control
	net.Conn
	tx *Trace
}

var _ net.Conn = &connTrace{}

// Read implements net.Conn.Read and saves network events.
func (c *connTrace) Read(b []byte) (int, error) {
	network := c.RemoteAddr().Network()
	addr := c.RemoteAddr().String()
	started := c.tx.Since(c.tx.ZeroTime)
	count, err := c.Conn.Read(b)
	finished := c.tx.Since(c.tx.ZeroTime)
	select {
	case c.tx.NetworkEvent <- NewArchivalNetworkEvent(
		c.tx.Index, started, netxlite.ReadOperation, network, addr, count, err, finished):
	default: // buffer is full
	}
	return count, err
}

// Write implements net.Conn.Write and saves network events.
func (c *connTrace) Write(b []byte) (int, error) {
	network := c.RemoteAddr().Network()
	addr := c.RemoteAddr().String()
	started := c.tx.Since(c.tx.ZeroTime)
	count, err := c.Conn.Write(b)
	finished := c.tx.Since(c.tx.ZeroTime)
	select {
	case c.tx.NetworkEvent <- NewArchivalNetworkEvent(
		c.tx.Index, started, netxlite.WriteOperation, network, addr, count, err, finished):
	default: // buffer is full
	}
	return count, err
}

// NewArchivalNetworkEvent creates a new model.ArchivalNetworkEvent.
func NewArchivalNetworkEvent(index int64, started time.Duration, operation string, network string,
	address string, count int, err error, finished time.Duration) *model.ArchivalNetworkEvent {
	return &model.ArchivalNetworkEvent{
		Address:   address,
		Failure:   tracex.NewFailure(err),
		NumBytes:  int64(count),
		Operation: operation,
		Proto:     network,
		T:         finished.Seconds(),
		Tags:      []string{},
	}
}

// NewAnnotationArchivalNetworkEvent is a simplified NewArchivalNetworkEvent
// where we create a simple annotation without attached I/O info.
func NewAnnotationArchivalNetworkEvent(
	index int64, time time.Duration, operation string) *model.ArchivalNetworkEvent {
	return NewArchivalNetworkEvent(index, time, operation, "", "", 0, nil, time)
}

// NetworkEvents drains the network events buffered inside the NetworkEvent channel.
func (tx *Trace) NetworkEvents() (out []*model.ArchivalNetworkEvent) {
	for {
		select {
		case ev := <-tx.NetworkEvent:
			out = append(out, ev)
		default:
			return // done
		}
	}
}
