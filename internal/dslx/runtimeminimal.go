package dslx

import (
	"io"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewMinimalRuntime creates a minimal [Runtime] implementation.
//
// This [Runtime] implementation does not collect any [*Observations].
func NewMinimalRuntime() *MinimalRuntime {
	return &MinimalRuntime{
		mu: sync.Mutex{},
		v:  []io.Closer{},
	}
}

var _ Runtime = &MinimalRuntime{}

// MinimalRuntime is a minimal [Runtime] implementation.
type MinimalRuntime struct {
	mu sync.Mutex
	v  []io.Closer
}

// MaybeTrackConn implements Runtime.
func (p *MinimalRuntime) MaybeTrackConn(conn io.Closer) {
	if conn != nil {
		defer p.mu.Unlock()
		p.mu.Lock()
		p.v = append(p.v, conn)
	}
}

// Close implements Runtime.
func (p *MinimalRuntime) Close() error {
	// Implementation note: reverse order is such that we close TLS
	// connections before we close the TCP connections they use. Hence
	// we'll _gracefully_ close TLS connections.
	defer p.mu.Unlock()
	p.mu.Lock()
	for idx := len(p.v) - 1; idx >= 0; idx-- {
		_ = p.v[idx].Close()
	}
	p.v = nil // reset
	return nil
}

// NewTrace implements Runtime.
func (p *MinimalRuntime) NewTrace(index int64, zeroTime time.Time, tags ...string) Trace {
	return &minimalTrace{idx: index, tags: tags, zt: zeroTime}
}

type minimalTrace struct {
	idx  int64
	tags []string
	zt   time.Time
}

// CloneBytesReceivedMap implements Trace.
func (tx *minimalTrace) CloneBytesReceivedMap() (out map[string]int64) {
	return make(map[string]int64)
}

// DNSLookupsFromRoundTrip implements Trace.
func (tx *minimalTrace) DNSLookupsFromRoundTrip() (out []*model.ArchivalDNSLookupResult) {
	return []*model.ArchivalDNSLookupResult{}
}

// Index implements Trace.
func (tx *minimalTrace) Index() int64 {
	return tx.idx
}

// NetworkEvents implements Trace.
func (tx *minimalTrace) NetworkEvents() (out []*model.ArchivalNetworkEvent) {
	return []*model.ArchivalNetworkEvent{}
}

// NewDialerWithoutResolver implements Trace.
func (tx *minimalTrace) NewDialerWithoutResolver(dl model.DebugLogger, wrappers ...model.DialerWrapper) model.Dialer {
	return netxlite.NewDialerWithoutResolver(dl, wrappers...)
}

// NewParallelUDPResolver implements Trace.
func (tx *minimalTrace) NewParallelUDPResolver(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver {
	return netxlite.NewParallelUDPResolver(logger, dialer, address)
}

// NewQUICDialerWithoutResolver implements Trace.
func (tx *minimalTrace) NewQUICDialerWithoutResolver(listener model.UDPListener, dl model.DebugLogger, wrappers ...model.QUICDialerWrapper) model.QUICDialer {
	return netxlite.NewQUICDialerWithoutResolver(listener, dl, wrappers...)
}

// NewStdlibResolver implements Trace.
func (tx *minimalTrace) NewStdlibResolver(logger model.DebugLogger) model.Resolver {
	return netxlite.NewStdlibResolver(logger)
}

// NewTLSHandshakerStdlib implements Trace.
func (tx *minimalTrace) NewTLSHandshakerStdlib(dl model.DebugLogger) model.TLSHandshaker {
	return netxlite.NewTLSHandshakerStdlib(dl)
}

// QUICHandshakes implements Trace.
func (tx *minimalTrace) QUICHandshakes() (out []*model.ArchivalTLSOrQUICHandshakeResult) {
	return []*model.ArchivalTLSOrQUICHandshakeResult{}
}

// TCPConnects implements Trace.
func (tx *minimalTrace) TCPConnects() (out []*model.ArchivalTCPConnectResult) {
	return []*model.ArchivalTCPConnectResult{}
}

// TLSHandshakes implements Trace.
func (tx *minimalTrace) TLSHandshakes() (out []*model.ArchivalTLSOrQUICHandshakeResult) {
	return []*model.ArchivalTLSOrQUICHandshakeResult{}
}

// Tags implements Trace.
func (tx *minimalTrace) Tags() []string {
	return tx.tags
}

// TimeSince implements Trace.
func (tx *minimalTrace) TimeSince(t0 time.Time) time.Duration {
	return time.Since(t0)
}

// ZeroTime implements Trace.
func (tx *minimalTrace) ZeroTime() time.Time {
	return tx.zt
}
