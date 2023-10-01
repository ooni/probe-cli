package pdsl

import (
	"io"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Runtime is the DSL execution environment.
type Runtime interface {
	// Close closes connections registered using RegisterClose.
	Close() error

	// Logger returns the logger to use.
	Logger() model.Logger

	// NewTrace creates a new [Trace].
	NewTrace(traceID int64, zeroTime time.Time, tags ...string) Trace

	// NewTraceID returns a new unique trace ID.
	NewTraceID() int64

	// RegisterCloser remebers to close the given connection.
	RegisterCloser(conn io.Closer)

	// ZeroTime returns the reference measurement time.
	ZeroTime() time.Time
}
