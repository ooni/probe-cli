package ntor

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// fetchTargets fetches the tor experiment's targets.
func (m *Measurer) fetchTargets(
	ctx context.Context, sess model.ExperimentSession) (
	map[string]model.TorTarget, error) {
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return sess.FetchTorTargets(ctx, sess.ProbeCC())
}
