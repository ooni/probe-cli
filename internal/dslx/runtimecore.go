package dslx

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Runtime is the runtime in which we execute the DSL.
type Runtime interface {
	// Close closes all the connection tracked using MaybeTrackConn.
	Close() error

	// IDGenerator returns an atomic counter used to generate
	// separate unique IDs for each trace.
	IDGenerator() *atomic.Int64

	// Logger returns the base logger to use.
	Logger() model.Logger

	// MaybeTrackConn tracks a connection such that it is closed
	// when you call the Runtime's Close method.
	MaybeTrackConn(conn io.Closer)

	// NewTrace creates a [Trace] instance. Note that each [Runtime]
	// creates its own [Trace] type. A [Trace] is not guaranteed to collect
	// [*Observations]. For example, [NewMinimalRuntime] creates a [Runtime]
	// that does not collect any [*Observations].
	NewTrace(index int64, zeroTime time.Time, tags ...string) Trace

	// Observations returns the [*Observations] saved so far and clears our
	// internal copy such that the next call to this method only returns
	// the [*Observations] saved since the previous call.
	//
	// You can safely call this method from multiple goroutine contexts.
	Observations() *Observations

	// SaveObservations saves [*Observations] inside the [Runtime]. You can
	// get the saved [*Observations] by calling Observations.
	//
	// You can safely call this method from multiple goroutine contexts.
	SaveObservations(obs ...*Observations)

	// ZeroTime returns the runtime's "zero" time, which is used as the
	// starting point to generate observation's delta times.
	ZeroTime() time.Time
}
