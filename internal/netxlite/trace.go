package netxlite

//
// Context-based tracing
//

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// traceKey is the private type used to set/retrieve the context's trace.
type traceKey struct{}

// ContextTraceOrDefault retrieves the trace bound to the context or returns
// a default implementation of the trace in case no tracing was configured.
func ContextTraceOrDefault(ctx context.Context) model.Trace {
	t, _ := ctx.Value(traceKey{}).(model.Trace)
	return traceOrDefault(t)
}

// ContextWithTrace returns a new context that binds to the given trace. If the
// given trace is nil, this function will call panic.
func ContextWithTrace(ctx context.Context, trace model.Trace) context.Context {
	runtimex.PanicIfTrue(trace == nil, "netxlite.WithTrace passed a nil trace")
	return context.WithValue(ctx, traceKey{}, trace)
}

// traceOrDefault takes in input a trace and returns in output the
// given trace, if not nil, or a default trace implementation.
func traceOrDefault(trace model.Trace) model.Trace {
	if trace != nil {
		return trace
	}
	return &traceDefault{}
}

// traceDefault is a default model.Trace implementation where each method is a no-op.
type traceDefault struct{}

var _ model.Trace = &traceDefault{}

// TimeNow implements model.Trace.TimeNow
func (*traceDefault) TimeNow() time.Time {
	return time.Now()
}

func (*traceDefault) OnDNSRoundTripForLookupHost(started time.Time, reso model.Resolver, query model.DNSQuery,
	response model.DNSResponse, addrs []string, err error, finished time.Time) {
	// nothing
}

// OnConnectDone implements model.Trace.OnConnectDone.
func (*traceDefault) OnConnectDone(
	started time.Time, network, domain, remoteAddr string, err error, finished time.Time) {
	// nothing
}

// OnTLSHandshakeStart implements model.Trace.OnTLSHandshakeStart.
func (*traceDefault) OnTLSHandshakeStart(now time.Time, remoteAddr string, config *tls.Config) {
	// nothing
}

// OnTLSHandshakeDone implements model.Trace.OnTLSHandshakeDone.
func (*traceDefault) OnTLSHandshakeDone(started time.Time, remoteAddr string, config *tls.Config,
	state tls.ConnectionState, err error, finished time.Time) {
	// nothing
}
