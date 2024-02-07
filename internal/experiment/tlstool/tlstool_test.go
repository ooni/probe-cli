package tlstool_test

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/tlstool"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := tlstool.NewExperimentMeasurer(tlstool.Config{})
	if measurer.ExperimentName() != "tlstool" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestRunWithExplicitSNI(t *testing.T) {
	ctx := context.Background()
	measurer := tlstool.NewExperimentMeasurer(tlstool.Config{
		SNI: "dns.google",
	})
	measurement := new(model.Measurement)
	measurement.Input = "8.8.8.8:853"
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     &mockable.Session{},
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunWithImplicitSNI(t *testing.T) {
	ctx := context.Background()
	measurer := tlstool.NewExperimentMeasurer(tlstool.Config{})
	measurement := new(model.Measurement)
	measurement.Input = "dns.google:853"
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     &mockable.Session{},
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cause failure
	measurer := tlstool.NewExperimentMeasurer(tlstool.Config{})
	measurement := new(model.Measurement)
	measurement.Input = "dns.google:853"
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     &mockable.Session{},
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}
