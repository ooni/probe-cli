package measure

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// TODO(bassosimone): add a unique ID to each new connection
// so we can be sure the events depend on that conn.

// Trace contains a network events trace.
type Trace struct {
	begin   time.Time
	entries []*TraceEntry
	mu      sync.Mutex
}

// NewTrace creates a new trace.
func NewTrace(begin time.Time) *Trace {
	return &Trace{begin: begin}
}

func (t *Trace) elapsed() time.Duration {
	return time.Since(t.begin)
}

// TraceEntry is an entry inside of a trace.
type TraceEntry struct {
	// Operation is the operation name.
	Operation string `json:"operation"`

	// Address is the remote endpoint address.
	Address string `json:"address"`

	// Started is when we started.
	Started time.Duration `json:"started"`

	// Completed is when we were done.
	Completed time.Duration `json:"completed"`

	// Failure is the error that occurred.
	Failure error `json:"failure"`

	// NumBytes is the number of bytes transferred.
	NumBytes int `json:"num_bytes"`
}

// ExtractEvents extracs the trace events leaving the trace empty.
func (t *Trace) ExtractEvents() (o []*TraceEntry) {
	t.mu.Lock()
	o = t.entries
	t.entries = nil
	t.mu.Unlock()
	return
}

// dial dials a connection and wraps it.
func (t *Trace) dial(ctx context.Context, dialer netxlite.Dialer,
	network, address string) (net.Conn, error) {
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &traceConn{Conn: conn, trace: t}, nil
}

func (t *Trace) add(op string, addr string,
	count int, err error, t0, t1 time.Duration) {
	t.mu.Lock()
	t.entries = append(t.entries, &TraceEntry{
		Operation: op,
		Address:   addr,
		Started:   t0,
		Completed: t1,
		Failure:   err,
		NumBytes:  count,
	})
	t.mu.Unlock()
}

func (t *Trace) safeAddrString(addr net.Addr) (s string) {
	if addr != nil {
		s = addr.String()
	}
	return
}

type traceConn struct {
	net.Conn
	trace *Trace
}

func (c *traceConn) Read(b []byte) (int, error) {
	addr := c.trace.safeAddrString(c.Conn.RemoteAddr())
	t0 := c.trace.elapsed()
	count, err := c.Conn.Read(b)
	t1 := c.trace.elapsed()
	c.trace.add("read", addr, count, err, t0, t1)
	return count, err
}

func (c *traceConn) Write(b []byte) (int, error) {
	addr := c.trace.safeAddrString(c.Conn.RemoteAddr())
	t0 := c.trace.elapsed()
	count, err := c.Conn.Write(b)
	t1 := c.trace.elapsed()
	c.trace.add("write", addr, count, err, t0, t1)
	return count, err
}

func (t *Trace) wrapQUICListener(ql netxlite.QUICListener) netxlite.QUICListener {
	return &traceQUICListener{QUICListener: ql, trace: t}
}

type traceQUICListener struct {
	netxlite.QUICListener
	trace *Trace
}

func (ql *traceQUICListener) Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	conn, err := ql.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	return &traceUDPLikeConn{UDPLikeConn: conn, trace: ql.trace}, nil
}

type traceUDPLikeConn struct {
	quicx.UDPLikeConn
	trace *Trace
}

func (c *traceUDPLikeConn) ReadFrom(p []byte) (int, net.Addr, error) {
	t0 := c.trace.elapsed()
	count, addr, err := c.UDPLikeConn.ReadFrom(p)
	t1 := c.trace.elapsed()
	c.trace.add("read_from", c.trace.safeAddrString(addr), count, err, t0, t1)
	return count, addr, err
}

func (c *traceUDPLikeConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	t0 := c.trace.elapsed()
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	t1 := c.trace.elapsed()
	c.trace.add("write_to", c.trace.safeAddrString(addr), count, err, t0, t1)
	return count, err
}
