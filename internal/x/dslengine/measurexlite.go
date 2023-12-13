package dslengine

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

// NewRuntimeMeasurexLite creates a [Runtime] using [measurexlite] to collect [*Observations].
func NewRuntimeMeasurexLite(logger model.Logger, zeroTime time.Time, options ...Option) *RuntimeMeasurexLite {
	values := newOptionValues(options...)

	rt := &RuntimeMeasurexLite{
		MinimalRuntime: NewMinimalRuntime(logger, zeroTime, options...),
		netx:           values.netx,
	}

	return rt
}

// RuntimeMeasurexLite uses [measurexlite] to collect [*Observations.]
type RuntimeMeasurexLite struct {
	*MinimalRuntime
	netx model.MeasuringNetwork
}

// NewTrace implements Runtime.
func (p *RuntimeMeasurexLite) NewTrace(index int64, zeroTime time.Time, tags ...string) dslvm.Trace {
	trace := measurexlite.NewTrace(index, zeroTime, tags...)
	trace.Netx = p.netx
	return trace
}

var _ dslvm.Runtime = &RuntimeMeasurexLite{}
