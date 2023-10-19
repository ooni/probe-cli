// Package progress contains utilities to emit progress.
package progress

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Scaler implements [model.ExperimentCallbacks] and scales progress
// as instructed through the [NewScaler] constructor.
//
// The [*Scaler] is safe to use from multiple goroutine contexts.
type Scaler struct {
	cbs    model.ExperimentCallbacks
	offset float64
	total  float64
}

// NewScaler constructs a new [*Scaler] using the given offset and total
// and emitting progress using the given [model.ExperimentCallbacks].
//
// The offset is added to each progress value we emit. The total is
// used to scale the 100% to a suitable subset.
//
// For example, with offset equal to 0.1 and total equal to 0.5, the value
// 0.5 corresponds to 0.3 and the value 1 (i.e., 100%) is 0.5.
//
// This func PANICS if offset<0, offset >= total, total<=0, total>1.
func NewScaler(callbacks model.ExperimentCallbacks, offset, total float64) *Scaler {
	runtimex.Assert(offset >= 0.0 && offset < total, "NewScaler: offset must be >= 0 and < total")
	runtimex.Assert(total > 0.0 && total <= 1, "NewScaler: total must be > 0 and <= 1")
	return &Scaler{
		cbs:    callbacks,
		offset: offset,
		total:  total,
	}
}

var _ model.ExperimentCallbacks = &Scaler{}

// OnProgress implements model.ExperimentCallbacks.
func (s *Scaler) OnProgress(percentage float64, message string) {
	s.cbs.OnProgress(s.offset+percentage*(s.total-s.offset), message)
}
