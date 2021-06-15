package ntor

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// measure measures each target and returns all the measurements.
func (m *Measurer) measure(ctx context.Context,
	targets map[string]model.TorTarget) map[string]TargetResults {
	timeout := time.Duration(len(targets)) * 15 * time.Second // proportional
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return m.doMeasure(ctx, targets)
}

// doMeasure implements measure.
func (m *Measurer) doMeasure(ctx context.Context,
	targets map[string]model.TorTarget) map[string]TargetResults {
	out := make(map[string]TargetResults)
	for name, info := range targets {
		out[name] = TargetResults{
			TargetAddress:  info.Address,
			TargetName:     name,
			TargetProtocol: info.Protocol,
			TargetSource:   info.Source,
		}
	}
	return out
}
