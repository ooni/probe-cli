package mocks

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestExperimentBuilder(t *testing.T) {
	t.Run("Interruptible", func(t *testing.T) {
		eb := &ExperimentBuilder{
			MockInterruptible: func() bool {
				return true
			},
		}
		if !eb.Interruptible() {
			t.Fatal("unexpected value")
		}
	})

	t.Run("InputPolicy", func(t *testing.T) {
		eb := &ExperimentBuilder{
			MockInputPolicy: func() model.InputPolicy {
				return model.InputOrQueryBackend
			},
		}
		if eb.InputPolicy() != model.InputOrQueryBackend {
			t.Fatal("unexpected value")
		}
	})

	t.Run("Options", func(t *testing.T) {
		expected := errors.New("mocked error")
		eb := &ExperimentBuilder{
			MockOptions: func() (map[string]model.ExperimentOptionInfo, error) {
				return nil, expected
			},
		}
		out, err := eb.Options()
		if !errors.Is(err, expected) {
			t.Fatal("unexpected value")
		}
		if len(out) > 0 {
			t.Fatal("unexpected value")
		}
	})

	t.Run("SetOptionAny", func(t *testing.T) {
		expected := errors.New("mocked error")
		eb := &ExperimentBuilder{
			MockSetOptionAny: func(key string, value any) error {
				return expected
			},
		}
		err := eb.SetOptionAny("antani", 1245678)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected value")
		}
	})

	t.Run("SetOptionsAny", func(t *testing.T) {
		expected := errors.New("mocked error")
		eb := &ExperimentBuilder{
			MockSetOptionsAny: func(options map[string]any) error {
				return expected
			},
		}
		err := eb.SetOptionsAny(make(map[string]any))
		if !errors.Is(err, expected) {
			t.Fatal("unexpected value")
		}
	})

	t.Run("SetCallbacks", func(t *testing.T) {
		var called bool
		eb := &ExperimentBuilder{
			MockSetCallbacks: func(callbacks model.ExperimentCallbacks) {
				called = true
			},
		}
		eb.SetCallbacks(model.NewPrinterCallbacks(model.DiscardLogger))
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("NewExperiment", func(t *testing.T) {
		exp := &Experiment{}
		eb := &ExperimentBuilder{
			MockNewExperiment: func() model.Experiment {
				return exp
			},
		}
		if out := eb.NewExperiment(); out != exp {
			t.Fatal("invalid result")
		}
	})
}
