package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestExperimentBuilderEngineWebConnectivity(t *testing.T) {
	// create a session for testing that does not use the network at all
	sess := newSessionForTestingNoLookups(t)

	// create an experiment builder for Web Connectivity
	builder, err := sess.NewExperimentBuilder("WebConnectivity")
	if err != nil {
		t.Fatal(err)
	}

	// create suitable loader config
	config := &model.ExperimentTargetLoaderConfig{
		CheckInConfig: &model.OOAPICheckInConfig{
			// nothing
		},
		Session:      sess,
		StaticInputs: nil,
		SourceFiles:  nil,
	}

	// create the loader
	loader := builder.NewTargetLoader(config)

	// create cancelled context to interrupt immediately so that we
	// don't use the network when running this test
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// attempt to load targets
	targets, err := loader.Load(ctx)

	// make sure we've got the expected error
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected err", err)
	}

	// make sure there are no targets
	if len(targets) != 0 {
		t.Fatal("expected zero length targets")
	}
}
