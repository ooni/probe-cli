package echcheck

import (
	"context"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"testing"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "echcheck" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected version")
	}
}

func TestMeasurerMeasureWithInvalidInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel the context
	measurer := NewExperimentMeasurer(Config{})
	measurement := &model.Measurement{
		Input: "http://example.org",
	}
	err := measurer.Run(
		ctx,
		newsession(),
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestMeasurerMeasureWithInvalidInput2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel the context
	measurer := NewExperimentMeasurer(Config{})
	measurement := &model.Measurement{
		// leading space to test url.Parse failure
		Input: " https://example.org",
	}
	err := measurer.Run(
		ctx,
		newsession(),
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestMeasurementSuccess(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	err := measurer.Run(
		context.Background(),
		newsession(),
		&model.Measurement{},
		model.NewPrinterCallbacks(log.Log),
	)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	summary, err := measurer.GetSummaryKeys(&model.Measurement{})

	if summary.(SummaryKeys).IsAnomaly != false {
		t.Fatal("expected false")
	}
}

func newsession() model.ExperimentSession {
	return &mockable.Session{MockableLogger: log.Log}
}
