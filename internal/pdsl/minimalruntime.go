package pdsl

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewMinimalRuntime creates a minimal runtime that does not collect OONI observations.
func NewMinimalRuntime(logger model.Logger) Runtime {
	return &minimalRuntime{
		cl:     []io.Closer{},
		logger: logger,
		mu:     sync.Mutex{},
	}
}

type minimalRuntime struct {
	cl     []io.Closer
	logger model.Logger
	mu     sync.Mutex
}

// Close implements Runtime.
func (rt *minimalRuntime) Close() error {
	defer rt.mu.Unlock()
	rt.mu.Lock()
	for idx := len(rt.cl) - 1; idx >= 0; idx-- {
		_ = rt.cl[idx].Close()
	}
	rt.cl = []io.Closer{}
	return nil
}

// Logger implements Runtime.
func (rt *minimalRuntime) Logger() model.Logger {
	return rt.logger
}

// NewTrace implements Runtime.
func (rt *minimalRuntime) NewTrace(traceID int64, zeroTime time.Time, tags ...string) Trace {
	return &netxlite.Netx{Underlying: nil}
}

var idgen = &atomic.Int64{}

// NewTraceID implements Runtime.
func (rt *minimalRuntime) NewTraceID() int64 {
	return idgen.Add(1)
}

// RegisterCloser implements Runtime.
func (rt *minimalRuntime) RegisterCloser(closer io.Closer) {
	rt.mu.Lock()
	rt.cl = append(rt.cl, closer)
	rt.mu.Unlock()
}

var zeroTime = time.Now()

// ZeroTime implements Runtime.
func (rt *minimalRuntime) ZeroTime() time.Time {
	return zeroTime
}
