package measurexlite

//
// Definition of Trace
//

import (
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
	// Index is the unique index of this trace within the
	// current measurement. Note that this field MUST be read-only. Writing it
	// once you have constructed a trace MAY lead to data races.
	Index int64

	// Netx is the network to use for measuring. The constructor inits this
	// field using a [*netxlite.Netx]. You MAY override this field for testing. Make
	// sure you do that before you start measuring to avoid data races.
	Netx model.MeasuringNetwork

	// bytesReceivedMap maps a remote host with the bytes we received
	// from such a remote host. Accessing this map requires one to
	// additionally hold the bytesReceivedMu mutex.
	bytesReceivedMap map[string]int64

	// bytesReceivedMu protects the bytesReceivedMap from concurrent
	// access from multiple goroutines.
	bytesReceivedMu *sync.Mutex

	// dnsLookup is MANDATORY and buffers DNS Lookup observations.
	dnsLookup chan *model.ArchivalDNSLookupResult

	// delayedDNSResponse is MANDATORY and buffers delayed DNS responses.
	delayedDNSResponse chan *model.ArchivalDNSLookupResult

	// networkEvent is MANDATORY and buffers network events.
	networkEvent chan *model.ArchivalNetworkEvent

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
		Index:            index,
		Netx:             &netxlite.Netx{Underlying: nil}, // use the host network
		bytesReceivedMap: make(map[string]int64),
		bytesReceivedMu:  &sync.Mutex{},
		dnsLookup: make(
			chan *model.ArchivalDNSLookupResult,
			DNSLookupBufferSize,
		),
		delayedDNSResponse: make(
			chan *model.ArchivalDNSLookupResult,
			DelayedDNSResponseBufferSize,
		),
		networkEvent: make(
			chan *model.ArchivalNetworkEvent,
			NetworkEventBufferSize,
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
