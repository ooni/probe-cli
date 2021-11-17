package psiphon_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/psiphon"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// Implementation note: integration test performed by
// the $topdir/experiment_test.go file

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := psiphon.NewExperimentMeasurer(psiphon.Config{})
	if measurer.ExperimentName() != "psiphon" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.5.1" {
		t.Fatal("unexpected version")
	}
}

func TestRunWithCancelledContext(t *testing.T) {
	measurer := psiphon.NewExperimentMeasurer(psiphon.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	measurement := new(model.Measurement)
	err := measurer.Run(ctx, newfakesession(), measurement,
		model.NewPrinterCallbacks(log.Log))
	if !errors.Is(err, context.Canceled) {
		t.Fatal("expected another error here")
	}
	tk := measurement.TestKeys.(*psiphon.TestKeys)
	if tk.MaxRuntime <= 0 {
		t.Fatal("you did not set the max runtime")
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(psiphon.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestRunWithCustomInputAndCancelledContext(t *testing.T) {
	expected := "http://x.org"
	measurement := &model.Measurement{
		Input: model.MeasurementTarget(expected),
	}
	measurer := psiphon.NewExperimentMeasurer(psiphon.Config{})
	measurer.(*psiphon.Measurer).BeforeGetHook = func(g urlgetter.Getter) {
		if g.Target != expected {
			t.Fatal("target was not correctly set")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	err := measurer.Run(ctx, newfakesession(), measurement,
		model.NewPrinterCallbacks(log.Log))
	if !errors.Is(err, context.Canceled) {
		t.Fatal("expected another error here")
	}
	tk := measurement.TestKeys.(*psiphon.TestKeys)
	if tk.MaxRuntime <= 0 {
		t.Fatal("you did not set the max runtime")
	}
}

func TestRunWillPrintSomethingWithCancelledContext(t *testing.T) {
	measurement := new(model.Measurement)
	measurer := psiphon.NewExperimentMeasurer(psiphon.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	measurer.(*psiphon.Measurer).BeforeGetHook = func(g urlgetter.Getter) {
		time.Sleep(2 * time.Second)
		cancel() // fail after we've given the printer a chance to run
	}
	observer := observerCallbacks{progress: &atomicx.Int64{}}
	err := measurer.Run(ctx, newfakesession(), measurement, observer)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("expected another error here")
	}
	tk := measurement.TestKeys.(*psiphon.TestKeys)
	if tk.MaxRuntime <= 0 {
		t.Fatal("you did not set the max runtime")
	}
	if observer.progress.Load() < 2 {
		t.Fatal("not enough progress emitted?!")
	}
}

type observerCallbacks struct {
	progress *atomicx.Int64
}

func (d observerCallbacks) OnProgress(percentage float64, message string) {
	d.progress.Add(1)
}

func newfakesession() model.ExperimentSession {
	return &mockable.Session{MockableLogger: log.Log}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &psiphon.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysGood(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &psiphon.TestKeys{TestKeys: urlgetter.TestKeys{
		BootstrapTime: 123,
	}}}
	m := &psiphon.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(psiphon.SummaryKeys)
	if sk.BootstrapTime != 123 {
		t.Fatal("invalid latency")
	}
	if sk.Failure != "" {
		t.Fatal("invalid failure")
	}
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysFailure(t *testing.T) {
	expected := io.EOF.Error()
	measurement := &model.Measurement{TestKeys: &psiphon.TestKeys{TestKeys: urlgetter.TestKeys{
		BootstrapTime: 123,
		Failure:       &expected,
	}}}
	m := &psiphon.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(psiphon.SummaryKeys)
	if sk.BootstrapTime != 123 {
		t.Fatal("invalid latency")
	}
	if sk.Failure != expected {
		t.Fatal("invalid failure")
	}
	if sk.IsAnomaly == false {
		t.Fatal("invalid isAnomaly")
	}
}
