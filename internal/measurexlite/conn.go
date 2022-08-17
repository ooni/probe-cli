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
	started := c.tx.TimeSince(c.tx.ZeroTime)
	count, err := c.Conn.Read(b)
	finished := c.tx.TimeSince(c.tx.ZeroTime)
	select {
	case c.tx.networkEvent <- NewArchivalNetworkEvent(
		c.tx.Index, started, netxlite.ReadOperation, network, addr, count, err, finished):
	default: // buffer is full
	}
	return count, err
}

// Write implements net.Conn.Write and saves network events.
func (c *connTrace) Write(b []byte) (int, error) {
	network := c.RemoteAddr().Network()
	addr := c.RemoteAddr().String()
	started := c.tx.TimeSince(c.tx.ZeroTime)
	count, err := c.Conn.Write(b)
	finished := c.tx.TimeSince(c.tx.ZeroTime)
	select {
	case c.tx.networkEvent <- NewArchivalNetworkEvent(
		c.tx.Index, started, netxlite.WriteOperation, network, addr, count, err, finished):
	default: // buffer is full
	}
	return count, err
}

// MaybeUDPLikeClose is a convenience function for closing a conn only when such a conn isn't nil.
func MaybeCloseUDPLikeConn(conn model.UDPLikeConn) (err error) {
	if conn != nil {
		err = conn.Close()
	}
	return
}

// WrapUDPLikeConn returns a wrapped conn that saves network events into this trace.
func (tx *Trace) WrapUDPLikeConn(conn model.UDPLikeConn) model.UDPLikeConn {
	return &udpLikeConnTrace{
		UDPLikeConn: conn,
		tx:          tx,
	}
}

// udpLikeConnTrace is a trace-aware model.UDPLikeConn.
type udpLikeConnTrace struct {
	// Implementation note: it seems ~safe to use embedding here because model.UDPLikeConn
	// contains fields deriving from how lucas-clemente/quic-go uses the standard library
	model.UDPLikeConn
	tx *Trace
}

// Read implements model.UDPLikeConn.ReadFrom and saves network events.
func (c *udpLikeConnTrace) ReadFrom(b []byte) (int, net.Addr, error) {
	started := c.tx.TimeSince(c.tx.ZeroTime)
	count, addr, err := c.UDPLikeConn.ReadFrom(b)
	finished := c.tx.TimeSince(c.tx.ZeroTime)
	address := addrStringIfNotNil(addr)
	select {
	case c.tx.networkEvent <- NewArchivalNetworkEvent(
		c.tx.Index, started, netxlite.ReadFromOperation, "udp", address, count, err, finished):
	default: // buffer is full
	}
	return count, addr, err
}

// Write implements model.UDPLikeConn.WriteTo and saves network events.
func (c *udpLikeConnTrace) WriteTo(b []byte, addr net.Addr) (int, error) {
	started := c.tx.TimeSince(c.tx.ZeroTime)
	address := addr.String()
	count, err := c.UDPLikeConn.WriteTo(b, addr)
	finished := c.tx.TimeSince(c.tx.ZeroTime)
	select {
	case c.tx.networkEvent <- NewArchivalNetworkEvent(
		c.tx.Index, started, netxlite.WriteToOperation, "udp", address, count, err, finished):
	default: // buffer is full
	}
	return count, err
}

// addrStringIfNotNil returns the string of the given addr
// unless the addr is nil, in which case it returns an empty string.
func addrStringIfNotNil(addr net.Addr) (out string) {
	if addr != nil {
		out = addr.String()
	}
	return
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
		case ev := <-tx.networkEvent:
			out = append(out, ev)
		default:
			return // done
		}
	}
}

// FirstNetworkEvent drains the network events buffered inside the NetworkEvents channel
// and returns the first NetworkEvent.
func (tx *Trace) FirstNetworkEvent() *model.ArchivalNetworkEvent {
	ev := tx.NetworkEvents()
	if len(ev) < 1 {
		return nil
	}
	return ev[0]
}
