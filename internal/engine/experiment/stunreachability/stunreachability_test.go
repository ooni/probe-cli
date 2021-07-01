package stunreachability_test

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/stunreachability"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/pion/stun"
)

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := stunreachability.NewExperimentMeasurer(stunreachability.Config{})
	if measurer.ExperimentName() != "stunreachability" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.2.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestRun(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// See https://github.com/ooni/probe-engine/issues/874#issuecomment-679850652
		t.Skip("skipping broken test on GitHub Actions")
	}
	measurer := stunreachability.NewExperimentMeasurer(stunreachability.Config{})
	measurement := new(model.Measurement)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*stunreachability.TestKeys)
	if tk.Failure != nil {
		t.Fatal("expected nil failure here")
	}
	if tk.Endpoint != "stun.l.google.com:19302" {
		t.Fatal("unexpected endpoint")
	}
	if len(tk.NetworkEvents) <= 0 {
		t.Fatal("no network events?!")
	}
	if len(tk.Queries) <= 0 {
		t.Fatal("no DNS queries?!")
	}
}

func TestRunCustomInput(t *testing.T) {
	input := "stun.ekiga.net:3478"
	measurer := stunreachability.NewExperimentMeasurer(stunreachability.Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget(input)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*stunreachability.TestKeys)
	if tk.Failure != nil {
		t.Fatal("expected nil failure here")
	}
	if tk.Endpoint != input {
		t.Fatal("unexpected endpoint")
	}
	if len(tk.NetworkEvents) <= 0 {
		t.Fatal("no network events?!")
	}
	if len(tk.Queries) <= 0 {
		t.Fatal("no DNS queries?!")
	}
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately fail everything
	measurer := stunreachability.NewExperimentMeasurer(stunreachability.Config{})
	measurement := new(model.Measurement)
	err := measurer.Run(
		ctx,
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if err.Error() != "interrupted" {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*stunreachability.TestKeys)
	if *tk.Failure != "interrupted" {
		t.Fatal("expected different failure here")
	}
	if tk.Endpoint != "stun.l.google.com:19302" {
		t.Fatal("unexpected endpoint")
	}
	if len(tk.NetworkEvents) <= 0 {
		t.Fatal("no network events?!")
	}
	if len(tk.Queries) <= 0 {
		t.Fatal("no DNS queries?!")
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(stunreachability.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestNewClientFailure(t *testing.T) {
	config := &stunreachability.Config{}
	expected := errors.New("mocked error")
	config.SetNewClient(
		func(conn stun.Connection, options ...stun.ClientOption) (*stun.Client, error) {
			return nil, expected
		})
	measurer := stunreachability.NewExperimentMeasurer(*config)
	measurement := new(model.Measurement)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*stunreachability.TestKeys)
	if !strings.HasPrefix(*tk.Failure, "unknown_failure") {
		t.Fatal("expected different failure here")
	}
	if tk.Endpoint != "stun.l.google.com:19302" {
		t.Fatal("unexpected endpoint")
	}
	if len(tk.NetworkEvents) <= 0 {
		t.Fatal("no network events?!")
	}
	if len(tk.Queries) <= 0 {
		t.Fatal("no DNS queries?!")
	}
}

func TestStartFailure(t *testing.T) {
	config := &stunreachability.Config{}
	expected := errors.New("mocked error")
	config.SetDialContext(
		func(ctx context.Context, network, address string) (net.Conn, error) {
			conn := &stunreachability.FakeConn{WriteError: expected}
			return conn, nil
		})
	measurer := stunreachability.NewExperimentMeasurer(*config)
	measurement := new(model.Measurement)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*stunreachability.TestKeys)
	if !strings.HasPrefix(*tk.Failure, "unknown_failure") {
		t.Fatal("expected different failure here")
	}
	if tk.Endpoint != "stun.l.google.com:19302" {
		t.Fatal("unexpected endpoint")
	}
	// We're bypassing normal network with custom dial function
	if len(tk.NetworkEvents) > 0 {
		t.Fatal("network events?!")
	}
	if len(tk.Queries) > 0 {
		t.Fatal("DNS queries?!")
	}
}

func TestReadFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	config := &stunreachability.Config{}
	expected := errors.New("mocked error")
	config.SetDialContext(
		func(ctx context.Context, network, address string) (net.Conn, error) {
			conn := &stunreachability.FakeConn{ReadError: expected}
			return conn, nil
		})
	measurer := stunreachability.NewExperimentMeasurer(*config)
	measurement := new(model.Measurement)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, stun.ErrTransactionTimeOut) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*stunreachability.TestKeys)
	if *tk.Failure != errorsx.FailureGenericTimeoutError {
		t.Fatal("expected different failure here")
	}
	if tk.Endpoint != "stun.l.google.com:19302" {
		t.Fatal("unexpected endpoint")
	}
	// We're bypassing normal network with custom dial function
	if len(tk.NetworkEvents) > 0 {
		t.Fatal("network events?!")
	}
	if len(tk.Queries) > 0 {
		t.Fatal("DNS queries?!")
	}
}

func TestSummaryKeysGeneric(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &stunreachability.TestKeys{}}
	m := &stunreachability.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(stunreachability.SummaryKeys)
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}
