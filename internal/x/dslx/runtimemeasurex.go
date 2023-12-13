package dslx

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// RuntimeMeasurexLiteOption is an option for initializing a [*RuntimeMeasurexLite].
type RuntimeMeasurexLiteOption func(rt *RuntimeMeasurexLite)

// RuntimeMeasurexLiteOptionMeasuringNetwork allows to configure which [model.MeasuringNetwork] to use.
func RuntimeMeasurexLiteOptionMeasuringNetwork(netx model.MeasuringNetwork) RuntimeMeasurexLiteOption {
	return func(rt *RuntimeMeasurexLite) {
		rt.netx = netx
	}
}

// NewRuntimeMeasurexLite creates a [Runtime] using [measurexlite] to collect [*Observations].
func NewRuntimeMeasurexLite(logger model.Logger, zeroTime time.Time, options ...RuntimeMeasurexLiteOption) *RuntimeMeasurexLite {
	rt := &RuntimeMeasurexLite{
		MinimalRuntime: NewMinimalRuntime(logger, zeroTime),
		netx:           &netxlite.Netx{Underlying: nil}, // implies using the host's network
	}
	for _, option := range options {
		option(rt)
	}
	return rt
}

// RuntimeMeasurexLite uses [measurexlite] to collect [*Observations.]
type RuntimeMeasurexLite struct {
	*MinimalRuntime
	netx model.MeasuringNetwork
}

// NewTrace implements Runtime.
func (p *RuntimeMeasurexLite) NewTrace(index int64, zeroTime time.Time, tags ...string) Trace {
	trace := measurexlite.NewTrace(index, zeroTime, tags...)
	trace.Netx = p.netx
	return trace
}

var _ Runtime = &RuntimeMeasurexLite{}
