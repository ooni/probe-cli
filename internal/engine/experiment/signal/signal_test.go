package signal_test

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := signal.NewExperimentMeasurer(signal.Config{})
	if measurer.ExperimentName() != "signal" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.2.2" {
		t.Fatal("unexpected version")
	}
}

func TestGood(t *testing.T) {
	measurer := signal.NewExperimentMeasurer(signal.Config{})
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*signal.TestKeys)
	if tk.Agent != "redirect" {
		t.Fatal("unexpected Agent")
	}
	if tk.FailedOperation != nil {
		t.Fatal("unexpected FailedOperation")
	}
	if tk.Failure != nil {
		t.Fatal("unexpected Failure")
	}
	if len(tk.NetworkEvents) <= 0 {
		t.Fatal("no NetworkEvents?!")
	}
	if len(tk.Queries) <= 0 {
		t.Fatal("no Queries?!")
	}
	if len(tk.Requests) <= 0 {
		t.Fatal("no Requests?!")
	}
	if len(tk.TCPConnect) <= 0 {
		t.Fatal("no TCPConnect?!")
	}
	if len(tk.TLSHandshakes) <= 0 {
		t.Fatal("no TLSHandshakes?!")
	}
	if tk.SignalBackendFailure != nil {
		t.Fatal("unexpected SignalBackendFailure")
	}
	if tk.SignalBackendStatus != "ok" {
		t.Fatal("unexpected SignalBackendStatus")
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(signal.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestUpdate(t *testing.T) {
	tk := signal.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "GET"},
			Target: "https://textsecure-service.whispersystems.org/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	if tk.SignalBackendStatus != "blocked" {
		t.Fatal("SignalBackendStatus should be blocked")
	}
	if *tk.SignalBackendFailure != netxlite.FailureEOFError {
		t.Fatal("invalid SignalBackendError")
	}
}

func TestBadSignalCA(t *testing.T) {
	measurer := signal.NewExperimentMeasurer(signal.Config{
		SignalCA: "INVALIDCA",
	})
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	}
	err := measurer.Run(context.Background(), args)
	if err.Error() != "AppendCertsFromPEM failed" {
		t.Fatal("not the error we expected")
	}
}

func TestGetSummaryInvalidType(t *testing.T) {
	measurer := signal.Measurer{}
	in := make(chan int)
	out, err := measurer.GetSummaryKeys(&model.Measurement{TestKeys: in})
	if err == nil || err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil output here")
	}
}
