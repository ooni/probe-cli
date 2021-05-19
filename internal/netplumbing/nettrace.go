package netplumbing

import (
	"context"
	"sync"
)

const (
	// TraceKindConnect identifies a trace collected during connect.
	TraceKindConnect = "connect"

	// TraceKindDNSRoundTrip is a trace collected during the DNS round trip.
	TraceKindDNSRoundTrip = "dns_round_trip"

	// TraceKindHTTPRoundTrip is a trace collected during the HTTP round trip.
	TraceKindHTTPRoundTrip = "http_round_trip"

	// TraceKindReadFrom identifies a trace collected during read_from.
	TraceKindReadFrom = "read_from"

	// TraceKindRead identifies a trace collected during read.
	TraceKindRead = "read"

	// TraceKindResolve identifies a trace collected during resolve.
	TraceKindResolve = "resolve"

	// TraceKindQUICHandshake identifies a trace collected during a QUIC handshake.
	TraceKindQUICHandshake = "quic_handshake"

	// TraceKindTLSHandshake identifies a trace collected during a TLS handshake.
	TraceKindTLSHandshake = "tls_handshake"

	// TraceKindWriteTo identifies a trace collected during writeTo.
	TraceKindWriteTo = "write_to"

	// TraceKindWrite identifies a trace collected during write.
	TraceKindWrite = "write"
)

// TraceEvent is an event occurred when tracing.
type TraceEvent interface {
	// Kind returns the event kind.
	Kind() string
}

// TraceHeader is the header for a list of related traces. To
// collect traces, create a TraceHeader and bind it to a context
// using the netplumbing.WithTrace function.
//
// To obtain the current TraceHeader from a context, use the
// netplumbing.ContextTraceHeader function.
//
// Calling WithTrace multiple times creates a stack of headers
// such that ContextTraceHeader only returns the top most header
// to the caller. Using this pattern, you can collect traces
// in concurrent code. Then you can join/merge the traces using
// the TraceHeader.MoveOut to extract the traces.
type TraceHeader struct {
	// events contains the events collected so far.
	events []TraceEvent

	// mu provides mutual exclusion.
	mu sync.Mutex
}

// add adds an event to the trace.
func (tr *TraceHeader) add(ev TraceEvent) {
	defer tr.mu.Unlock()
	tr.mu.Lock()
	tr.events = append(tr.events, ev)
}

// MoveOut moves the collected events out of the trace.
func (tr *TraceHeader) MoveOut() []TraceEvent {
	defer tr.mu.Unlock()
	tr.mu.Lock()
	out := tr.events
	tr.events = nil
	return out
}

// traceHeaderKey identifies a TraceHeader among context values.
type traceHeaderKey struct{}

// WithTraceHeader creates a copy of the current context
// that is using the given TraceHeader.
func WithTraceHeader(ctx context.Context, th *TraceHeader) context.Context {
	if th == nil {
		panic("netplumbing: passed a nil TraceHeader")
	}
	return context.WithValue(ctx, traceHeaderKey{}, th)
}

// ContextTraceHeader returns the currently configured TraceHeader or nil.
func ContextTraceHeader(ctx context.Context) *TraceHeader {
	th, _ := ctx.Value(traceHeaderKey{}).(*TraceHeader)
	return th
}
