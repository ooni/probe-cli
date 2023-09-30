// Package dslminimalruntime implements a minimal DSL runtime.
//
// This runtime does not collect OONI observations. You MAY want to use this
// runtime when you're not interested into producing OONI measurements.
package dslminimalruntime

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/dslmodel"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// New creates a minimal runtime that does not collect OONI observations.
func New(logger model.Logger) dslmodel.Runtime {
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

// Close implements dslmodel.Runtime.
func (rt *minimalRuntime) Close() error {
	defer rt.mu.Unlock()
	rt.mu.Lock()
	for idx := len(rt.cl) - 1; idx >= 0; idx-- {
		_ = rt.cl[idx].Close()
	}
	rt.cl = []io.Closer{}
	return nil
}

// Logger implements dslmodel.Runtime.
func (rt *minimalRuntime) Logger() model.Logger {
	return rt.logger
}

// NewTrace implements dslmodel.Runtime.
func (rt *minimalRuntime) NewTrace(traceID int64, zeroTime time.Time, tags ...string) dslmodel.Trace {
	return &netxlite.Netx{Underlying: nil}
}

var idgen = &atomic.Int64{}

// NewTraceID implements dslmodel.Runtime.
func (rt *minimalRuntime) NewTraceID() int64 {
	return idgen.Add(1)
}

// RegisterCloser implements dslmodel.Runtime.
func (rt *minimalRuntime) RegisterCloser(closer io.Closer) {
	rt.mu.Lock()
	rt.cl = append(rt.cl, closer)
	rt.mu.Unlock()
}

var zeroTime = time.Now()

// ZeroTime implements dslmodel.Runtime.
func (rt *minimalRuntime) ZeroTime() time.Time {
	return zeroTime
}
