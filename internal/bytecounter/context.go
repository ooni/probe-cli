package bytecounter

//
// Implicit byte counting based on context
//

import (
	"context"
	"net"
)

type byteCounterSessionKey struct{}

// ContextSessionByteCounter retrieves the session byte counter from the context
func ContextSessionByteCounter(ctx context.Context) *Counter {
	counter, _ := ctx.Value(byteCounterSessionKey{}).(*Counter)
	return counter
}

// WithSessionByteCounter assigns the session byte counter to the context.
func WithSessionByteCounter(ctx context.Context, counter *Counter) context.Context {
	return context.WithValue(ctx, byteCounterSessionKey{}, counter)
}

type byteCounterExperimentKey struct{}

// ContextExperimentByteCounter retrieves the experiment byte counter from the context
func ContextExperimentByteCounter(ctx context.Context) *Counter {
	counter, _ := ctx.Value(byteCounterExperimentKey{}).(*Counter)
	return counter
}

// WithExperimentByteCounter assigns the experiment byte counter to the context.
func WithExperimentByteCounter(ctx context.Context, counter *Counter) context.Context {
	return context.WithValue(ctx, byteCounterExperimentKey{}, counter)
}

// MaybeWrapWithContextByteCounters wraps a conn with the byte counters
// that have previosuly been configured into a context.
func MaybeWrapWithContextByteCounters(ctx context.Context, conn net.Conn) net.Conn {
	conn = MaybeWrapConn(conn, ContextExperimentByteCounter(ctx))
	conn = MaybeWrapConn(conn, ContextSessionByteCounter(ctx))
	return conn
}
