package echcheck

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
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

// qaenv creates a [netemx.QAEnv] with a single example.org test server and a DoH server.
func qaenv() *netemx.QAEnv {
	cfg := []*netemx.ScenarioDomainAddresses{
		{
			Domain:    "example.org",
			Addresses: []string{"130.192.91.7"},
			Role:      netemx.ScenarioRoleExampleLikeWebServer,
		},
		{
			Domain:    "mozilla.cloudflare-dns.com",
			Addresses: []string{"130.192.91.13"},
			Role:      netemx.ScenarioRoleDNSOverHTTPS,
		},
	}
	return netemx.MustNewScenario(cfg)
}

func TestMeasurerMeasureWithCancelledContext(t *testing.T) {
	// create QAEnv
	env := qaenv()
	defer env.Close()

	env.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel the context

		// create measurer
		measurer := NewExperimentMeasurer(Config{})
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: &model.Measurement{},
			Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		}
		// run measurement
		err := measurer.Run(ctx, args)
		if err == nil {
			t.Fatal("expected an error here")
		}
		if err.Error() != "interrupted" {
			t.Fatal("unexpected error type")
		}
	})

}

func TestMeasurerMeasureWithInvalidInput(t *testing.T) {
	// create QAEnv
	env := qaenv()
	defer env.Close()

	// create measurer
	measurer := NewExperimentMeasurer(Config{})
	args := &model.ExperimentArgs{
		Callbacks: model.NewPrinterCallbacks(model.DiscardLogger),
		Measurement: &model.Measurement{
			// leading space to test url.Parse failure
			Input: " https://example.org",
		},
		Session: &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
	}
	// run measurement
	err := measurer.Run(context.Background(), args)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "input is not an URL" {
		t.Fatal("unexpected error type")
	}
}

func TestMeasurementSuccess(t *testing.T) {
	env := qaenv()
	defer env.Close()

	env.Do(func() {
		measurer := NewExperimentMeasurer(Config{})
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: &model.Measurement{},
			Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		}

		err := measurer.Run(context.Background(), args)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		summary, err := measurer.GetSummaryKeys(&model.Measurement{})
		if err != nil {
			t.Fatal("unexpected error GetSummaryKeys", err)
		}
		if summary.(SummaryKeys).IsAnomaly != false {
			t.Fatal("expected false")
		}
	})
}
