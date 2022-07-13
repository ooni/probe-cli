package oonirun

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// TODO(bassosimone): it would be cool to write unit tests. However, to do that
// we need to ~redesign the engine package for unit-testability.

func newSession(ctx context.Context, t *testing.T) *engine.Session {
	config := engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{},
		KVStore:                &kvstore.Memory{},
		Logger:                 model.DiscardLogger,
		ProxyURL:               nil,
		SoftwareName:           "miniooni",
		SoftwareVersion:        version.Version,
		TempDir:                os.TempDir(),
		TorArgs:                []string{},
		TorBinary:              "",
		TunnelDir:              "",
	}
	sess, err := engine.NewSession(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	return sess
}

func TestExperimentRunWithExample(t *testing.T) {
	ctx := context.Background()
	desc := &Experiment{
		Annotations: map[string]string{
			"platform": "linux",
		},
		ExtraOptions: map[string]any{
			"SleepTime": int64(10 * time.Millisecond),
		},
		Inputs:         []string{},
		InputFilePaths: []string{},
		MaxRuntime:     0,
		Name:           "example",
		NoCollector:    true,
		NoJSON:         true,
		Random:         false,
		ReportFile:     "",
		Session:        newSession(ctx, t),
	}
	if err := desc.Run(ctx); err != nil {
		t.Fatal(err)
	}
}
