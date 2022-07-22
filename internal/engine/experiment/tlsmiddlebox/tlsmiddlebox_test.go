package tlsmiddlebox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TODO(DecFox): Add more experiment-specific tests

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "tlsmiddlebox" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestConfig_maxttl(t *testing.T) {
	c := Config{}
	if c.maxttl() != 20 {
		t.Fatal("invalid default number of repetitions")
	}
}

func TestConfig_delay(t *testing.T) {
	c := Config{}
	if c.delay() != 100*time.Millisecond {
		t.Fatal("invalid default delay")
	}
}

func TestConfig_resolver(t *testing.T) {
	c := Config{}
	if c.resolverURL() != "https://mozilla.cloudflare-dns.com/dns-query" {
		t.Fatal("invalid resolver URL")
	}
}

func TestConfig_snipass(t *testing.T) {
	c := Config{}
	if c.snicontrol() != "example.com" {
		t.Fatal("invalid pass SNI")
	}
}

func TestConfig_sni(t *testing.T) {
	t.Run("without config", func(t *testing.T) {
		c := Config{}
		if c.sni("example.com") != "example.com" {
			t.Fatal("invalid sni")
		}
	})
	t.Run("with config", func(t *testing.T) {
		c := Config{
			SNI: "google.com",
		}
		if c.sni("example.com") != "google.com" {
			t.Fatal("invalid sni")
		}
	})
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
