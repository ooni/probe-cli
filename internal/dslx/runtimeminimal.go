package dslx

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// MinimalRuntimeOption is an option for configuring the [*MinimalRuntime].
type MinimalRuntimeOption func(rt *MinimalRuntime)

// MinimalRuntimeOptionMeasuringNetwork configures the [model.MeasuringNetwork] to use.
func MinimalRuntimeOptionMeasuringNetwork(netx model.MeasuringNetwork) MinimalRuntimeOption {
	return func(rt *MinimalRuntime) {
		rt.netx = netx
	}
}

// NewMinimalRuntime creates a minimal [Runtime] implementation.
//
// This [Runtime] implementation does not collect any [*Observations].
func NewMinimalRuntime(logger model.Logger, zeroTime time.Time, options ...MinimalRuntimeOption) *MinimalRuntime {
	rt := &MinimalRuntime{
		idg:    &atomic.Int64{},
		logger: logger,
		mu:     sync.Mutex{},
		netx:   &netxlite.Netx{Underlying: nil}, // implies using the host's network
		ob:     NewObservations(),
		v:      []io.Closer{},
		zeroT:  zeroTime,
	}
	for _, option := range options {
		option(rt)
	}
	return rt
}

var _ Runtime = &MinimalRuntime{}

// MinimalRuntime is a minimal [Runtime] implementation.
type MinimalRuntime struct {
	idg    *atomic.Int64
	logger model.Logger
	mu     sync.Mutex
	netx   model.MeasuringNetwork
	ob     *Observations
	v      []io.Closer
	zeroT  time.Time
}

// Observations implements Runtime.
func (p *MinimalRuntime) Observations() *Observations {
	defer p.mu.Unlock()
	p.mu.Lock()
	o := p.ob
	p.ob = NewObservations()
	return o
}

// SaveObservations implements Runtime.
func (p *MinimalRuntime) SaveObservations(obs ...*Observations) {
	defer p.mu.Unlock()
	p.mu.Lock()
	for _, o := range obs {
		p.ob.NetworkEvents = append(p.ob.NetworkEvents, o.NetworkEvents...)
		p.ob.QUICHandshakes = append(p.ob.QUICHandshakes, o.QUICHandshakes...)
		p.ob.Queries = append(p.ob.Queries, o.Queries...)
		p.ob.Requests = append(p.ob.Requests, o.Requests...)
		p.ob.TCPConnect = append(p.ob.TCPConnect, o.TCPConnect...)
		p.ob.TLSHandshakes = append(p.ob.TLSHandshakes, o.TLSHandshakes...)
	}
}

// IDGenerator implements Runtime.
func (p *MinimalRuntime) IDGenerator() *atomic.Int64 {
	return p.idg
}

// Logger implements Runtime.
func (p *MinimalRuntime) Logger() model.Logger {
	return p.logger
}

// ZeroTime implements Runtime.
func (p *MinimalRuntime) ZeroTime() time.Time {
	return p.zeroT
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
	return &minimalTrace{idx: index, netx: p.netx, tags: tags, zt: zeroTime}
}

type minimalTrace struct {
	idx  int64
	netx model.MeasuringNetwork
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
	return tx.netx.NewDialerWithoutResolver(dl, wrappers...)
}

// NewParallelUDPResolver implements Trace.
func (tx *minimalTrace) NewParallelUDPResolver(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver {
	return tx.netx.NewParallelUDPResolver(logger, dialer, address)
}

// NewQUICDialerWithoutResolver implements Trace.
func (tx *minimalTrace) NewQUICDialerWithoutResolver(listener model.UDPListener, dl model.DebugLogger, wrappers ...model.QUICDialerWrapper) model.QUICDialer {
	return tx.netx.NewQUICDialerWithoutResolver(listener, dl, wrappers...)
}

// NewStdlibResolver implements Trace.
func (tx *minimalTrace) NewStdlibResolver(logger model.DebugLogger) model.Resolver {
	return tx.netx.NewStdlibResolver(logger)
}

// NewTLSHandshakerStdlib implements Trace.
func (tx *minimalTrace) NewTLSHandshakerStdlib(dl model.DebugLogger) model.TLSHandshaker {
	return tx.netx.NewTLSHandshakerStdlib(dl)
}

// NewUDPListener implements Trace
func (tx *minimalTrace) NewUDPListener() model.UDPListener {
	return tx.netx.NewUDPListener()
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
