package dnsping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

func TestConfig_domains(t *testing.T) {
	c := Config{}
	if c.domains() != "edge-chat.instagram.com example.com" {
		t.Fatal("invalid default domains list")
	}
}

func TestConfig_repetitions(t *testing.T) {
	c := Config{}
	if c.repetitions() != 10 {
		t.Fatal("invalid default number of repetitions")
	}
}

func TestConfig_delay(t *testing.T) {
	c := Config{}
	if c.delay() != time.Second {
		t.Fatal("invalid default delay")
	}
}

func TestMeasurer_run(t *testing.T) {
	// expectedPings is the expected number of pings
	const expectedPings = 4

	// runHelper is an helper function to run this set of tests.
	runHelper := func(input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			Domains:     "example.com",
			Delay:       1, // millisecond
			Repetitions: expectedPings,
		})
		if m.ExperimentName() != "dnsping" {
			t.Fatal("invalid experiment name")
		}
		if m.ExperimentVersion() != "0.3.0" {
			t.Fatal("invalid experiment version")
		}
		ctx := context.Background()
		meas := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		sess := &mocks.Session{
			MockLogger: func() model.Logger { return model.DiscardLogger },
		}
		callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
		args := &model.ExperimentArgs{
			Callbacks:   callbacks,
			Measurement: meas,
			Session:     sess,
		}
		err := m.Run(ctx, args)
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
		_, _, err := runHelper("https://8.8.8.8:443/")
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with missing port", func(t *testing.T) {
		_, _, err := runHelper("udp://8.8.8.8")
		if !errors.Is(err, errMissingPort) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with netem: without DPI: expect success", func(t *testing.T) {
		// we use the same empty DNS config for client and servers here
		dnsConfig := netem.NewDNSConfig()
		// create configuration for DNS server
		dnsConfig.AddRecord(
			"example.com",
			"example.com", // CNAME
			"93.184.216.34",
		)

		clientConf := &netemx.ClientConfig{
			DNSConfig:    dnsConfig,
			ResolverAddr: "8.8.8.8",
		}

		// create a new test environment
		env := netemx.NewEnvironment(clientConf, &netemx.ServersConfig{})
		defer env.Close()
		env.Do(func() {
			meas, m, err := runHelper("udp://8.8.8.8:53")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			tk, _ := (meas.TestKeys).(*TestKeys)
			if len(tk.Pings) != expectedPings*2 { // account for A & AAAA pings
				t.Fatal("unexpected number of pings", len(tk.Pings))
			}

			ask, err := m.GetSummaryKeys(meas)
			if err != nil {
				t.Fatal("cannot obtain summary")
			}
			summary := ask.(SummaryKeys)
			if summary.IsAnomaly {
				t.Fatal("expected no anomaly")
			}

			for _, p := range tk.Pings {
				if p.Query == nil {
					t.Fatal("QUery should not be nil")
				}
				if p.Query.Answers == nil {
					t.Fatal("p.Query.Answers should not be nil")
				}
				if p.Query.QueryType == "A" && p.Query.Failure != nil {
					t.Fatal("unexpected error", *p.Query.Failure)
				}
			}
		})
	})

	t.Run("with netem: with DPI that drops TCP segments to 8.8.8.8:443: expect failure", func(t *testing.T) {
		// we use the same empty DNS config for client and servers here
		dnsConfig := netem.NewDNSConfig()
		// create configuration for DNS server
		dnsConfig.AddRecord(
			"example.com",
			"example.com", // CNAME
			"93.184.216.34",
		)

		clientConf := &netemx.ClientConfig{
			DNSConfig:    dnsConfig,
			ResolverAddr: "8.8.8.8",
		}

		// create a new test environment
		env := netemx.NewEnvironment(clientConf, &netemx.ServersConfig{})
		defer env.Close()

		// add DPI engine to emulate the censorship condition
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
			Logger:          model.DiscardLogger,
			ServerIPAddress: "8.8.8.8",
			ServerPort:      53,
			ServerProtocol:  layers.IPProtocolUDP,
		})

		env.Do(func() {
			meas, m, err := runHelper("udp://8.8.8.8:53")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			tk, _ := (meas.TestKeys).(*TestKeys)

			// note: this experiment does not set anomaly but we still want
			// to have a test here for when we possibly will
			ask, err := m.GetSummaryKeys(meas)
			if err != nil {
				t.Fatal("cannot obtain summary")
			}
			summary := ask.(SummaryKeys)
			if summary.IsAnomaly {
				t.Fatal("expected no anomaly")
			}

			for _, p := range tk.Pings {
				if p.Query == nil {
					t.Fatal("Query should not be nil")
				}
				if p.Query.Answers != nil {
					t.Fatal("unexpected answers")
				}
				if p.Query.Failure == nil {
					t.Fatal("expected a failure here")
				}
			}
		})
	})
}
