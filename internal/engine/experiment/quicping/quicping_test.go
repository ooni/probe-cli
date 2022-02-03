package quicping_test

import (
	"context"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/quicping"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"net"
	"strings"
	"testing"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := quicping.NewExperimentMeasurer(quicping.Config{})
	if measurer.ExperimentName() != "quicping" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected version")
	}
}

func TestInvalidHost(t *testing.T) {
	measurer := quicping.NewExperimentMeasurer(quicping.Config{
		Port: int64(443),
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("a.a.a.a")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err == nil {
		t.Fatal("expected an error here")
	}
	if _, ok := err.(*net.DNSError); !ok {
		t.Fatal("unexpected error type")
	}
}

func TestReadTimeout(t *testing.T) {
	measurer := quicping.NewExperimentMeasurer(quicping.Config{
		Port:        int64(443),
		Timeout:     int64(10),
		Repetitions: int64(2),
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("cloudflare.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	tk := measurement.TestKeys.(*quicping.TestKeys)
	for i, ping := range tk.Pings {
		if ping.Failure == nil {
			t.Fatal("ping should have failed", i)
		}
		if !strings.Contains(*ping.Failure, "timeout") {
			t.Fatal("ping: unexpected error type", i, *ping.Failure)
		}
	}
}

func TestSucess(t *testing.T) {
	measurer := quicping.NewExperimentMeasurer(quicping.Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("cloudflare.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err != nil {
		t.Fatal("did not expect an error here")
	}
	tk := measurement.TestKeys.(*quicping.TestKeys)
	if tk.Domain != "cloudflare.com" {
		t.Fatal("unexpected domain")
	}
	if tk.Repetitions != 10 {
		t.Fatal("unexpected number of repetitions, default is 10")
	}
	if tk.Pings == nil || len(tk.Pings) != 10 {
		t.Fatal("not enough pings")
	}
	for i, ping := range tk.Pings {
		if ping.Failure != nil {
			t.Fatal("ping failed unexpectedly", i, *ping.Failure)
		}
		if ping.SupportedVersions == nil || len(ping.SupportedVersions) == 0 {
			t.Fatal("server did not respond with supported versions")
		}
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(quicping.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}
