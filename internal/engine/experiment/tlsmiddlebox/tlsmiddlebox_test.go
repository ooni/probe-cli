package tlsmiddlebox

// add more tests for tlsmiddlebox

import (
	"context"
	"testing"
	"time"

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

func TestConfig_iterations(t *testing.T) {
	c := Config{}
	if c.iterations() != 20 {
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
	if c.snipass() != "google.com" {
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

func TestMeasurer_Run(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewExperimentMeasurer(Config{
			SNI: "1337x.be",
		})
		ctx := context.Background()
		meas := &model.Measurement{
			Input: model.MeasurementTarget("https://www.example.com"),
		}
		sess := &mockable.Session{
			MockableLogger: model.DiscardLogger,
		}
		callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
		err := m.Run(ctx, sess, meas, callbacks)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
	})
}
