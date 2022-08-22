package measurexlite

//
// Definition of Trace
//

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Trace implements model.Trace.
//
// The zero-value of this struct is invalid. To construct you should either
// fill all the fields marked as MANDATORY or use NewTrace.
//
// Buffered channels
//
// NewTrace uses reasonable buffer sizes for the channels used for collecting
// events. You should drain the channels used by this implementation after
// each operation you perform (i.e., we expect you to peform step-by-step
// measurements). If you want larger (or smaller) buffers, then you should
// construct this data type manually with the desired buffer sizes.
//
// We have convenience methods for extracting events from the buffered
// channels. Otherwise, you could read the channels directly. (In which
// case, remember to issue nonblocking channel reads because channels are
// never closed and they're just written when new events occur.)
type Trace struct {
	// Index is the MANDATORY unique index of this trace within the
	// current measurement. If you don't care about uniquely identifying
	// traces, you can use zero to indicate the "default" trace.
	Index int64

	// networkEvent is MANDATORY and buffers network events.
	networkEvent chan *model.ArchivalNetworkEvent

	// NewStdlibResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewStdlibResolver factory.
	NewStdlibResolverFn func(logger model.Logger) model.Resolver

	// NewParallelUDPResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewParallelUDPResolver factory.
	NewParallelUDPResolverFn func(logger model.Logger, dialer model.Dialer, address string) model.Resolver

	// NewParallelDNSOverHTTPSResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewParallelDNSOverHTTPSUDPResolver factory.
	NewParallelDNSOverHTTPSResolverFn func(logger model.Logger, URL string) model.Resolver

	// NewDialerWithoutResolverFn is OPTIONAL and can be used to override
	// calls to the netxlite.NewDialerWithoutResolver factory.
	NewDialerWithoutResolverFn func(dl model.DebugLogger) model.Dialer

	// NewTLSHandshakerStdlibFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewTLSHandshakerStdlib factory.
	NewTLSHandshakerStdlibFn func(dl model.DebugLogger) model.TLSHandshaker

	// NewDialerWithoutResolverFn is OPTIONAL and can be used to override
	// calls to the netxlite.NewQUICDialerWithoutResolver factory.
	NewQUICDialerWithoutResolverFn func(listener model.QUICListener, dl model.DebugLogger) model.QUICDialer

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

	// TimeNowFn is OPTIONAL and can be used to override calls to time.Now
	// to produce deterministic timing when testing.
	TimeNowFn func() time.Time

	// ZeroTime is the MANDATORY time when we started the current measurement.
	ZeroTime time.Time
}

const (
	// NetworkEventBufferSize is the buffer size for constructing
	// the Trace's networkEvent buffered channel.
	NetworkEventBufferSize = 64

	// DNSLookupBufferSize is the buffer size for constructing
	// the Trace's dnsLookup buffered channel.
	DNSLookupBufferSize = 8

	// DNSResponseBufferSize is the buffer size for constructing
	// the Trace's dnsDelayedResponse buffered channel.
	DelayedDNSResponseBufferSize = 8

	// TCPConnectBufferSize is the buffer size for constructing
	// the Trace's tcpConnect buffered channel.
	TCPConnectBufferSize = 8

	// TLSHandshakeBufferSize is the buffer for construcing
	// the Trace's tlsHandshake buffered channel.
	TLSHandshakeBufferSize = 8

	// QUICHandshakeBufferSize is the buffer for constructing
	// the Trace's quicHandshake buffered channel.
	QUICHandshakeBufferSize = 8
)

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
// - zeroTime is the time when we started the current measurement.
func NewTrace(index int64, zeroTime time.Time) *Trace {
	return &Trace{
		Index: index,
		networkEvent: make(
			chan *model.ArchivalNetworkEvent,
			NetworkEventBufferSize,
		),
		NewDialerWithoutResolverFn: nil, // use default
		NewTLSHandshakerStdlibFn:   nil, // use default
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
		TimeNowFn: nil, // use default
		ZeroTime:  zeroTime,
	}
}

// newStdlibResolver indirectly calls the passed netxlite.NewStdlibResolver
// thus allowing us to mock this function for testing
func (tx *Trace) newStdlibResolver(logger model.Logger) model.Resolver {
	if tx.NewStdlibResolverFn != nil {
		return tx.NewStdlibResolverFn(logger)
	}
	return netxlite.NewStdlibResolver(logger)
}

// newParallelUDPResolver indirectly calls the passed netxlite.NewParallerUDPResolver
// thus allowing us to mock this function for testing
func (tx *Trace) newParallelUDPResolver(logger model.Logger, dialer model.Dialer, address string) model.Resolver {
	if tx.NewParallelUDPResolverFn != nil {
		return tx.NewParallelUDPResolverFn(logger, dialer, address)
	}
	return netxlite.NewParallelUDPResolver(logger, dialer, address)
}

// newParallelDNSOverHTTPSResolver indirectly calls the passed netxlite.NewParallerDNSOverHTTPSResolver
// thus allowing us to mock this function for testing
func (tx *Trace) newParallelDNSOverHTTPSResolver(logger model.Logger, URL string) model.Resolver {
	if tx.NewParallelDNSOverHTTPSResolverFn != nil {
		return tx.NewParallelDNSOverHTTPSResolverFn(logger, URL)
	}
	return netxlite.NewParallelDNSOverHTTPSResolver(logger, URL)
}

// newDialerWithoutResolver indirectly calls netxlite.NewDialerWithoutResolver
// thus allowing us to mock this func for testing.
func (tx *Trace) newDialerWithoutResolver(dl model.DebugLogger) model.Dialer {
	if tx.NewDialerWithoutResolverFn != nil {
		return tx.NewDialerWithoutResolverFn(dl)
	}
	return netxlite.NewDialerWithoutResolver(dl)
}

// newTLSHandshakerStdlib indirectly calls netxlite.NewTLSHandshakerStdlib
// thus allowing us to mock this func for testing.
func (tx *Trace) newTLSHandshakerStdlib(dl model.DebugLogger) model.TLSHandshaker {
	if tx.NewTLSHandshakerStdlibFn != nil {
		return tx.NewTLSHandshakerStdlibFn(dl)
	}
	return netxlite.NewTLSHandshakerStdlib(dl)
}

// newWUICDialerWithoutResolver indirectly calls netxlite.NewQUICDialerWithoutResolver
// thus allowing us to mock this func for testing.
func (tx *Trace) newQUICDialerWithoutResolver(listener model.QUICListener, dl model.DebugLogger) model.QUICDialer {
	if tx.NewQUICDialerWithoutResolverFn != nil {
		return tx.NewQUICDialerWithoutResolverFn(listener, dl)
	}
	return netxlite.NewQUICDialerWithoutResolver(listener, dl)
}

// TimeNow implements model.Trace.TimeNow.
func (tx *Trace) TimeNow() time.Time {
	if tx.TimeNowFn != nil {
		return tx.TimeNowFn()
	}
	return time.Now()
}

// TimeSince is equivalent to Trace.TimeNow().Sub(t0).
func (tx *Trace) TimeSince(t0 time.Time) time.Duration {
	return tx.TimeNow().Sub(t0)
}

var _ model.Trace = &Trace{}
