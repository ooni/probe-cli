package dslx

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewRuntimeMeasurexLite creates a [Runtime] using [measurexlite] to collect [*Observations].
func NewRuntimeMeasurexLite(logger model.Logger, zeroTime time.Time) *RuntimeMeasurexLite {
	return &RuntimeMeasurexLite{
		MinimalRuntime: NewMinimalRuntime(logger, zeroTime),
	}
}

// RuntimeMeasurexLite uses [measurexlite] to collect [*Observations.]
type RuntimeMeasurexLite struct {
	*MinimalRuntime
}

// NewTrace implements Runtime.
func (p *RuntimeMeasurexLite) NewTrace(index int64, zeroTime time.Time, tags ...string) Trace {
	return measurexlite.NewTrace(index, zeroTime, tags...)
}

var _ Runtime = &RuntimeMeasurexLite{}
