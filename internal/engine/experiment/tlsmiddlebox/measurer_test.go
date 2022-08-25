package tlsmiddlebox

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "tlsmiddlebox" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestMeasurer_input_failure(t *testing.T) {
	runHelper := func(input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{})
		ctx := context.Background()
		meas := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		sess := &mockable.Session{
			MockableLogger: model.DiscardLogger,
		}
		callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
		err := m.Run(ctx, sess, meas, callbacks)
		return meas, m, err
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
		_, _, err := runHelper("http://8.8.8.8:443/")
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})
}

// TODO(DecFox): The tests here are mainly on-network. We want to keep less of these and
// replace with more "local" tests using filtering. This ensures speed and confidence in
// while testing
func TestMeasurer_run(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	ctx := context.Background()
	meas := &model.Measurement{
		Input: model.MeasurementTarget("https://www.google.com"),
	}
	sess := &mockable.Session{
		MockableLogger: model.DiscardLogger,
	}
	callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
	err := m.Run(ctx, sess, meas, callbacks)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
}

func TestMeasurer_run_with_config(t *testing.T) {
	m := NewExperimentMeasurer(Config{
		SNIControl: "google.com",
		SNI:        "1337x.be",
	})
	ctx := context.Background()
	meas := &model.Measurement{
		Input: model.MeasurementTarget("https://example.com"),
	}
	sess := &mockable.Session{
		MockableLogger: model.DiscardLogger,
	}
	callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
	err := m.Run(ctx, sess, meas, callbacks)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
}
