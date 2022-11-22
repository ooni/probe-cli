package bittorrent

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestMeasurer_run(t *testing.T) {
	// runHelper is an helper function to run this set of tests.
	runHelper := func(input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		measurer := NewExperimentMeasurer(Config{})
		ctx := context.Background()
		measurement := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		session := &mockable.Session{
			MockableLogger: model.DiscardLogger,
		}

		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: measurement,
			Session:     session,
		}

		err := measurer.Run(ctx, args)
		return measurement, measurer, err
	}

	t.Run("with empty input", func(t *testing.T) {
		_, _, err := runHelper("")
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := runHelper("\t")
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper("https://8.8.8.8:443/")
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})
}
