package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestExperimentBuilderBasicOperations(t *testing.T) {
	// create a session for testing that does not use the network at all
	sess := newSessionForTestingNoLookups(t)

	// create an experiment builder for example
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}

	// example should be interruptible
	t.Run("Interruptible", func(t *testing.T) {
		if !builder.Interruptible() {
			t.Fatal("example should be interruptible")
		}
	})

	// we expect to see the InputNone input policy
	t.Run("InputPolicy", func(t *testing.T) {
		if builder.InputPolicy() != model.InputNone {
			t.Fatal("unexpectyed input policy")
		}
	})

	// get the options and check whether they are what we expect
	t.Run("Options", func(t *testing.T) {
		options, err := builder.Options()
		if err != nil {
			t.Fatal(err)
		}
		expectOptions := map[string]model.ExperimentOptionInfo{
			"Message":     {Doc: "Message to emit at test completion", Type: "string", Value: "Good day from the example experiment!"},
			"ReturnError": {Doc: "Toogle to return a mocked error", Type: "bool", Value: false},
			"SleepTime":   {Doc: "Amount of time to sleep for in nanosecond", Type: "int64", Value: int64(1000000000)},
		}
		if diff := cmp.Diff(expectOptions, options); diff != "" {
			t.Fatal(diff)
		}
	})

	// we can set a specific existing option
	t.Run("SetOptionAny", func(t *testing.T) {
		if err := builder.SetOptionAny("Message", "foobar"); err != nil {
			t.Fatal(err)
		}
		options, err := builder.Options()
		if err != nil {
			t.Fatal(err)
		}
		expectOptions := map[string]model.ExperimentOptionInfo{
			"Message":     {Doc: "Message to emit at test completion", Type: "string", Value: "foobar"},
			"ReturnError": {Doc: "Toogle to return a mocked error", Type: "bool", Value: false},
			"SleepTime":   {Doc: "Amount of time to sleep for in nanosecond", Type: "int64", Value: int64(1000000000)},
		}
		if diff := cmp.Diff(expectOptions, options); diff != "" {
			t.Fatal(diff)
		}
	})

	// we can set all options at the same time
	t.Run("SetOptions", func(t *testing.T) {
		inputs := map[string]any{
			"Message":     "foobar",
			"ReturnError": true,
		}
		if err := builder.SetOptionsAny(inputs); err != nil {
			t.Fatal(err)
		}
		options, err := builder.Options()
		if err != nil {
			t.Fatal(err)
		}
		expectOptions := map[string]model.ExperimentOptionInfo{
			"Message":     {Doc: "Message to emit at test completion", Type: "string", Value: "foobar"},
			"ReturnError": {Doc: "Toogle to return a mocked error", Type: "bool", Value: true},
			"SleepTime":   {Doc: "Amount of time to sleep for in nanosecond", Type: "int64", Value: int64(1000000000)},
		}
		if diff := cmp.Diff(expectOptions, options); diff != "" {
			t.Fatal(diff)
		}
	})

	// we can set all options using JSON
	t.Run("SetOptionsJSON", func(t *testing.T) {
		inputs := json.RawMessage(`{
			"Message":     "foobar",
			"ReturnError": true
		}`)
		if err := builder.SetOptionsJSON(inputs); err != nil {
			t.Fatal(err)
		}
		options, err := builder.Options()
		if err != nil {
			t.Fatal(err)
		}
		expectOptions := map[string]model.ExperimentOptionInfo{
			"Message":     {Doc: "Message to emit at test completion", Type: "string", Value: "foobar"},
			"ReturnError": {Doc: "Toogle to return a mocked error", Type: "bool", Value: true},
			"SleepTime":   {Doc: "Amount of time to sleep for in nanosecond", Type: "int64", Value: int64(1000000000)},
		}
		if diff := cmp.Diff(expectOptions, options); diff != "" {
			t.Fatal(diff)
		}
	})

	// TODO(bassosimone): we could possibly add more checks here. I am not doing this
	// right now, because https://github.com/ooni/probe-cli/pull/1629 mostly cares about
	// providing input and the rest of the codebase did not change.
	//
	// Also, it would make sense to eventually merge experimentbuilder.go with the
	// ./internal/registry package, which also has coverage.
	//
	// In conclusion, our main objective for now is to make sure we don't screw the
	// pooch when setting options using the experiment builder.
}
