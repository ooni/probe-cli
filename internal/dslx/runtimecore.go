package dslx

import (
	"io"
	"time"
)

// Runtime is the runtime in which we execute the DSL.
type Runtime interface {
	// Close closes all the connection tracked using MaybeTrackConn.
	Close() error

	// MaybeTrackConn tracks a connection such that it is closed
	// when you call the Runtime's Close method.
	MaybeTrackConn(conn io.Closer)

	// NewTrace creates a [Trace] instance. Note that each [Runtime]
	// creates its own [Trace] type. A [Trace] is not guaranteed to collect
	// [*Observations]. For example, [NewMinimalRuntime] creates a [Runtime]
	// that does not collect any [*Observations].
	NewTrace(index int64, zeroTime time.Time, tags ...string) Trace
}
