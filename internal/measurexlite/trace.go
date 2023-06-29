package measurexlite

//
// Definition of Trace
//

import (
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

// Trace implements [model.Trace]. We use a [context.Context] to register ourselves
// as the [model.Trace] and we implement the [model.Trace] callbacks to route events
// into internal buffered channels as explained in by the [measurexlite] package
// documentation. The zero-value of this struct is invalid. To construct use [NewTrace].
//
// [NewTrace] uses reasonable buffer sizes for the channels used for collecting
// events. You should drain the channels used by this implementation after
// each operation you perform (that is, we expect you to peform [step-by-step
// measurements]). As mentioned in the [measurexlite] package documentation,
// there are several methods for extracting events from the [*Trace].
//
// [step-by-step measurements]: https://github.com/ooni/probe-cli/blob/master/docs/design/dd-003-step-by-step.md
type Trace struct {
	// BytesSent is the atomic counter of bytes sent so far for this trace. While it would
	// be safe to write this field, given that it is atomic, doing that would be very
	// unexpected. In fact, we exported this field to allow the user to atomically read
	// the amount of bytes sent so far by this trace and keep statistics.
	BytesSent *atomic.Int64

	// BytesReceived is like BytesSent but for the bytes received.
	BytesReceived *atomic.Int64

	// Index is the unique index of this trace within the
	// current measurement. Note that this field MUST be read-only. Writing it
	// once you have constructed a trace MAY lead to data races.
	Index int64

	// networkEvent is MANDATORY and buffers network events.
	networkEvent chan *model.ArchivalNetworkEvent

	// newStdlibResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewStdlibResolver factory.
	newStdlibResolverFn func(logger model.Logger) model.Resolver

	// newParallelUDPResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewParallelUDPResolver factory.
	newParallelUDPResolverFn func(logger model.Logger, dialer model.Dialer, address string) model.Resolver

	// newParallelDNSOverHTTPSResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewParallelDNSOverHTTPSUDPResolver factory.
	newParallelDNSOverHTTPSResolverFn func(logger model.Logger, URL string) model.Resolver

	// newDialerWithoutResolverFn is OPTIONAL and can be used to override
	// calls to the netxlite.NewDialerWithoutResolver factory.
	newDialerWithoutResolverFn func(dl model.DebugLogger) model.Dialer

	// newTLSHandshakerStdlibFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewTLSHandshakerStdlib factory.
	newTLSHandshakerStdlibFn func(dl model.DebugLogger) model.TLSHandshaker

	// newTLSHandshakerUTLSFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewTLSHandshakerUTLS factory.
	newTLSHandshakerUTLSFn func(dl model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker

	// NewDialerWithoutResolverFn is OPTIONAL and can be used to override
	// calls to the netxlite.NewQUICDialerWithoutResolver factory.
	newQUICDialerWithoutResolverFn func(listener model.QUICListener, dl model.DebugLogger) model.QUICDialer

	// dnsLookup is MANDATORY and buffers DNS Lookup observations.
	dnsLookup chan *model.ArchivalDNSLookupResult

	// delayedDNSResponse is MANDATORY and buffers delayed DNS responses.
	delayedDNSResponse chan *model.ArchivalDNSLookupResult

	// tcpConnect is MANDATORY and buffers TCP connect observations.
	tcpConnect chan *model.ArchivalTCPConnectResult

	// tlsHandshake is MANDATORY and buffers TLS handshake observations.
	tlsHandshake chan *model.ArchivalTLSOrQUICHandshakeResult

	// quicHandshake is MANDATORY and buffers QUIC handshake observations.
	quicHandshake chan *model.ArchivalTLSOrQUICHandshakeResult

	// tags contains OPTIONAL tags to tag measurements.
	tags []string

	// timeNowFn is OPTIONAL and can be used to override calls to time.Now
	// to produce deterministic timing when testing.
	timeNowFn func() time.Time

	// ZeroTime is the time when we started the current measurement. This field
	// MUST be read-only. Writing it once you have constructed the trace will
	// likely read to data races.
	ZeroTime time.Time
}

// NetworkEventBufferSize is the [*Trace] buffer size for network I/O events.
const NetworkEventBufferSize = 64

// DNSLookupBufferSize is the [*Trace] buffer size for DNS lookup events.
const DNSLookupBufferSize = 8

// DNSResponseBufferSize is the [*Trace] buffer size for delayed DNS responses events.
const DelayedDNSResponseBufferSize = 8

// TCPConnectBufferSize is the [*Trace] buffer size for TCP connect events.
const TCPConnectBufferSize = 8

// TLSHandshakeBufferSize is the [*Trace] buffer size for TLS handshake events.
const TLSHandshakeBufferSize = 8

// QUICHandshakeBufferSize is the [*Trace] buffer size for QUIC handshake events.
const QUICHandshakeBufferSize = 8

// NewTrace creates a new instance of Trace using default settings.
//
// We create buffered channels using as buffer sizes the constants that
// are also defined by this package.
//
// Arguments:
//
// - index is the unique index of this trace within the current measurement (use
// zero if you don't care about giving this trace a unique ID);
//
// - zeroTime is the time when we started the current measurement;
//
// - tags contains optional tags to mark the archival data formats specially (e.g.,
// to identify that some traces belong to some submeasurements).
func NewTrace(index int64, zeroTime time.Time, tags ...string) *Trace {
	return &Trace{
		BytesSent:     &atomic.Int64{},
		BytesReceived: &atomic.Int64{},
		Index:         index,
		networkEvent: make(
			chan *model.ArchivalNetworkEvent,
			NetworkEventBufferSize,
		),
		newStdlibResolverFn:               nil, // use default
		newParallelUDPResolverFn:          nil, // use default
		newParallelDNSOverHTTPSResolverFn: nil, // use default
		newDialerWithoutResolverFn:        nil, // use default
		newTLSHandshakerStdlibFn:          nil, // use default
		newTLSHandshakerUTLSFn:            nil, // use default
		newQUICDialerWithoutResolverFn:    nil, // use default
		dnsLookup: make(
			chan *model.ArchivalDNSLookupResult,
			DNSLookupBufferSize,
		),
		delayedDNSResponse: make(
			chan *model.ArchivalDNSLookupResult,
			DelayedDNSResponseBufferSize,
		),
		tcpConnect: make(
			chan *model.ArchivalTCPConnectResult,
			TCPConnectBufferSize,
		),
		tlsHandshake: make(
			chan *model.ArchivalTLSOrQUICHandshakeResult,
			TLSHandshakeBufferSize,
		),
		quicHandshake: make(
			chan *model.ArchivalTLSOrQUICHandshakeResult,
			QUICHandshakeBufferSize,
		),
		tags:      tags,
		timeNowFn: nil, // use default
		ZeroTime:  zeroTime,
	}
}

// newStdlibResolver indirectly calls the passed netxlite.NewStdlibResolver
// thus allowing us to mock this function for testing
func (tx *Trace) newStdlibResolver(logger model.Logger) model.Resolver {
	if tx.newStdlibResolverFn != nil {
		return tx.newStdlibResolverFn(logger)
	}
	return netxlite.NewStdlibResolver(logger)
}

// newParallelUDPResolver indirectly calls the passed netxlite.NewParallerUDPResolver
// thus allowing us to mock this function for testing
func (tx *Trace) newParallelUDPResolver(logger model.Logger, dialer model.Dialer, address string) model.Resolver {
	if tx.newParallelUDPResolverFn != nil {
		return tx.newParallelUDPResolverFn(logger, dialer, address)
	}
	return netxlite.NewParallelUDPResolver(logger, dialer, address)
}

// newParallelDNSOverHTTPSResolver indirectly calls the passed netxlite.NewParallerDNSOverHTTPSResolver
// thus allowing us to mock this function for testing
func (tx *Trace) newParallelDNSOverHTTPSResolver(logger model.Logger, URL string) model.Resolver {
	if tx.newParallelDNSOverHTTPSResolverFn != nil {
		return tx.newParallelDNSOverHTTPSResolverFn(logger, URL)
	}
	return netxlite.NewParallelDNSOverHTTPSResolver(logger, URL)
}

// newDialerWithoutResolver indirectly calls netxlite.NewDialerWithoutResolver
// thus allowing us to mock this func for testing.
func (tx *Trace) newDialerWithoutResolver(dl model.DebugLogger) model.Dialer {
	if tx.newDialerWithoutResolverFn != nil {
		return tx.newDialerWithoutResolverFn(dl)
	}
	return netxlite.NewDialerWithoutResolver(dl)
}

// newTLSHandshakerStdlib indirectly calls netxlite.NewTLSHandshakerStdlib
// thus allowing us to mock this func for testing.
func (tx *Trace) newTLSHandshakerStdlib(dl model.DebugLogger) model.TLSHandshaker {
	if tx.newTLSHandshakerStdlibFn != nil {
		return tx.newTLSHandshakerStdlibFn(dl)
	}
	return netxlite.NewTLSHandshakerStdlib(dl)
}

// newTLSHandshakerUTLS indirectly calls netxlite.NewTLSHandshakerUTLS
// thus allowing us to mock this func for testing.
func (tx *Trace) newTLSHandshakerUTLS(dl model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker {
	if tx.newTLSHandshakerUTLSFn != nil {
		return tx.newTLSHandshakerUTLSFn(dl, id)
	}
	return netxlite.NewTLSHandshakerUTLS(dl, id)
}

// newQUICDialerWithoutResolver indirectly calls netxlite.NewQUICDialerWithoutResolver
// thus allowing us to mock this func for testing.
func (tx *Trace) newQUICDialerWithoutResolver(listener model.QUICListener, dl model.DebugLogger) model.QUICDialer {
	if tx.newQUICDialerWithoutResolverFn != nil {
		return tx.newQUICDialerWithoutResolverFn(listener, dl)
	}
	return netxlite.NewQUICDialerWithoutResolver(listener, dl)
}

// TimeNow implements model.Trace.TimeNow.
func (tx *Trace) TimeNow() time.Time {
	if tx.timeNowFn != nil {
		return tx.timeNowFn()
	}
	return time.Now()
}

// TimeSince is equivalent to Trace.TimeNow().Sub(t0).
func (tx *Trace) TimeSince(t0 time.Time) time.Duration {
	return tx.TimeNow().Sub(t0)
}

// Tags returns a copy of the tags configured for this trace.
func (tx *Trace) Tags() []string {
	return copyAndNormalizeTags(tx.tags)
}

var _ model.Trace = &Trace{}
