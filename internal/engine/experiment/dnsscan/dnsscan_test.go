package dnsscan

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestConfig_resolver(t *testing.T) {
	c := Config{}
	if c.resolver() != "8.8.8.8:53" {
		t.Fatal("invalid default domains list")
	}
}

func TestMeasurer_run(t *testing.T) {
	// runHelper is an helper function to run this set of tests.
	runHelper := func(input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			Resolver: "udp://8.8.8.8:53",
		})
		if m.ExperimentName() != "dnsscan" {
			t.Fatal("invalid experiment name")
		}
		if m.ExperimentVersion() != "0.1.0" {
			t.Fatal("invalid experiment version")
		}
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

	t.Run("with valid input", func(t *testing.T) {
		meas, m, err := runHelper("example.com")
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		tk := meas.TestKeys.(*TestKeys)
		if len(tk.Queries) != 2 {
			t.Fatal("unexpected number of queries", len(tk.Queries))
		}
		ask, err := m.GetSummaryKeys(meas)
		if err != nil {
			t.Fatal("cannot obtain summary")
		}
		summary := ask.(SummaryKeys)
		if summary.IsAnomaly {
			t.Fatal("expected no anomaly")
		}
	})
}
