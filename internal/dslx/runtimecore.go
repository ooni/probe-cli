package dslx

import (
	"io"
)

// Runtime is the runtime in which we execute the DSL.
type Runtime interface {
	// Close closes all the connection tracked using MaybeTrackConn.
	Close() error

	// MaybeTrackConn tracks a connection such that it is closed
	// when you call the Runtime's Close method.
	MaybeTrackConn(conn io.Closer)
}
