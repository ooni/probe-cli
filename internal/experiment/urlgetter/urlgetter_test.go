package urlgetter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestMeasurer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := urlgetter.NewExperimentMeasurer(urlgetter.Config{})
	if m.ExperimentName() != "urlgetter" {
		t.Fatal("invalid experiment name")
	}
	if m.ExperimentVersion() != "0.2.0" {
		t.Fatal("invalid experiment version")
	}
	measurement := new(model.Measurement)
	measurement.Input = "https://www.google.com"
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     &mockable.Session{},
	}
	err := m.Run(ctx, args)
	if !errors.Is(err, nil) { // nil because we want to submit the measurement
		t.Fatal("not the error we expected")
	}
	if len(measurement.Extensions) != 6 {
		t.Fatal("not the expected number of extensions")
	}
	tk := measurement.TestKeys.(*urlgetter.TestKeys)
	if len(tk.DNSCache) != 0 {
		t.Fatal("not the DNSCache value we expected")
	}
}

func TestMeasurerDNSCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := urlgetter.NewExperimentMeasurer(urlgetter.Config{
		DNSCache: "dns.google 8.8.8.8 8.8.4.4",
	})
	if m.ExperimentName() != "urlgetter" {
		t.Fatal("invalid experiment name")
	}
	if m.ExperimentVersion() != "0.2.0" {
		t.Fatal("invalid experiment version")
	}
	measurement := new(model.Measurement)
	measurement.Input = "https://www.google.com"
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     &mockable.Session{},
	}
	err := m.Run(ctx, args)
	if !errors.Is(err, nil) { // nil because we want to submit the measurement
		t.Fatal("not the error we expected")
	}
	if len(measurement.Extensions) != 6 {
		t.Fatal("not the expected number of extensions")
	}
	tk := measurement.TestKeys.(*urlgetter.TestKeys)
	if len(tk.DNSCache) != 1 || tk.DNSCache[0] != "dns.google 8.8.8.8 8.8.4.4" {
		t.Fatal("invalid tk.DNSCache")
	}
}
