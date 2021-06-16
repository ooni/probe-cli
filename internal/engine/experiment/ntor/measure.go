package ntor

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// measure measures each target and returns all the measurements.
func (m *Measurer) measure(ctx context.Context, logger model.Logger,
	targets map[string]model.TorTarget) map[string]TargetResults {
	timeout := time.Duration(len(targets)) * 15 * time.Second // proportional
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return m.doMeasure(ctx, logger, targets)
}

// doMeasure performs the measurement using a measurement service.
func (m *Measurer) doMeasure(ctx context.Context, logger model.Logger,
	targets map[string]model.TorTarget) map[string]TargetResults {
	mctx := newService(ctx, logger)
	defer mctx.stop()
	go mctx.reader(targets)
	out := make(map[string]TargetResults)
	for len(out) < len(targets) {
		mout := <-mctx.output
		out[mout.results.TargetName] = mout.results
	}
	return out
}
