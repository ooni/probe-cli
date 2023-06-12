package portfiltering

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "portfiltering" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

// TODO(DecFox): Skip this test with -short in a future iteration.
func TestMeasurer_run(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	meas := &model.Measurement{}
	sess := &mocks.Session{
		MockLogger: func() model.Logger {
			return model.DiscardLogger
		},
	}
	callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
	ctx := context.Background()
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: meas,
		Session:     sess,
	}
	err := m.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	tk := meas.TestKeys.(*TestKeys)
	if len(tk.TCPConnect) != len(Ports) {
		t.Fatal("unexpected number of ports")
	}
	ask, err := m.GetSummaryKeys(meas)
	if err != nil {
		t.Fatal("cannot obtain summary")
	}
	summary := ask.(SummaryKeys)
	if summary.IsAnomaly {
		t.Fatal("expected no anomaly")
	}
}
