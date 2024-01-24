package dslvm

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Trace collects [*Observations] using tracing. Specific implementations
// of this interface may be engineered to collect no [*Observations] for
// efficiency (i.e., when you don't care about collecting [*Observations]
// but you still want to use this package).
type Trace interface {
	// CloneBytesReceivedMap returns a clone of the internal bytes received map. The key of the
	// map is a string following the "EPNT_ADDRESS PROTO" pattern where the "EPNT_ADDRESS" contains
	// the endpoint address and "PROTO" is "tcp" or "udp".
	CloneBytesReceivedMap() (out map[string]int64)

	// DNSLookupsFromRoundTrip returns all the DNS lookup results collected so far.
	DNSLookupsFromRoundTrip() (out []*model.ArchivalDNSLookupResult)

	// Index returns the unique index used by this trace.
	Index() int64

	// NewDialerWithoutResolver is equivalent to netxlite.Netx.NewDialerWithoutResolver
	// except that it returns a model.Dialer that uses this trace.
	//
	// Caveat: the dialer wrappers are there to implement the
	// model.MeasuringNetwork interface, but they're not used by this function.
	NewDialerWithoutResolver(dl model.DebugLogger, wrappers ...model.DialerWrapper) model.Dialer

	// NewParallelUDPResolver returns a possibly-trace-ware parallel UDP resolver
	NewParallelUDPResolver(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver

	// NewQUICDialerWithoutResolver is equivalent to
	// netxlite.Netx.NewQUICDialerWithoutResolver except that it returns a
	// model.QUICDialer that uses this trace.
	//
	// Caveat: the dialer wrappers are there to implement the
	// model.MeasuringNetwork interface, but they're not used by this function.
	NewQUICDialerWithoutResolver(listener model.UDPListener,
		dl model.DebugLogger, wrappers ...model.QUICDialerWrapper) model.QUICDialer

	// NewTLSHandshakerStdlib is equivalent to netxlite.Netx.NewTLSHandshakerStdlib
	// except that it returns a model.TLSHandshaker that uses this trace.
	NewTLSHandshakerStdlib(dl model.DebugLogger) model.TLSHandshaker

	// NetworkEvents returns all the network events collected so far.
	NetworkEvents() (out []*model.ArchivalNetworkEvent)

	// NewStdlibResolver returns a possibly-trace-ware system resolver.
	NewStdlibResolver(logger model.DebugLogger) model.Resolver

	// NewUDPListener implements model.MeasuringNetwork.
	NewUDPListener() model.UDPListener

	// QUICHandshakes collects all the QUIC handshake results collected so far.
	QUICHandshakes() (out []*model.ArchivalTLSOrQUICHandshakeResult)

	// TCPConnects collects all the TCP connect results collected so far.
	TCPConnects() (out []*model.ArchivalTCPConnectResult)

	// TLSHandshakes collects all the TLS handshake results collected so far.
	TLSHandshakes() (out []*model.ArchivalTLSOrQUICHandshakeResult)

	// Tags returns the trace tags.
	Tags() []string

	// TimeSince is equivalent to Trace.TimeNow().Sub(t0).
	TimeSince(t0 time.Time) time.Duration

	// ZeroTime returns the "zero" time of this trace.
	ZeroTime() time.Time
}
