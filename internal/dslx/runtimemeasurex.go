package dslx

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
)

// NewRuntimeMeasurexLite creates a [Runtime] using [measurexlite] to collect [*Observations].
func NewRuntimeMeasurexLite() *RuntimeMeasurexLite {
	return &RuntimeMeasurexLite{
		MinimalRuntime: NewMinimalRuntime(),
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
