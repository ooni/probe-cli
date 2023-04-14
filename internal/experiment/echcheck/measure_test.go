package echcheck

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "echcheck" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.1.1" {
		t.Fatal("unexpected version")
	}
}

func TestMeasurerMeasureWithInvalidInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel the context
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurer := NewExperimentMeasurer(Config{})
	measurement := &model.Measurement{
		Input: "http://example.org",
	}
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(
		ctx,
		args,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestMeasurerMeasureWithInvalidInput2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel the context
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurer := NewExperimentMeasurer(Config{})
	measurement := &model.Measurement{
		// leading space to test url.Parse failure
		Input: " https://example.org",
	}
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(
		ctx,
		args,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestMeasurementSuccess(t *testing.T) {
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurer := NewExperimentMeasurer(Config{})
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: &model.Measurement{},
		Session:     sess,
	}
	err := measurer.Run(
		context.Background(),
		args,
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
