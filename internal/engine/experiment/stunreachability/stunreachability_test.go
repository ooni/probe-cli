package stunreachability

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/pion/stun"
)

const (
	defaultEndpoint = "stun.ekiga.net:3478"
	defaultInput    = "stun://" + defaultEndpoint
)

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "stunreachability" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.3.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestRunWithoutInput(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	measurement := new(model.Measurement)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, errStunMissingInput) {
		t.Fatal("not the error we expected", err)
	}
}

func TestRunWithInvalidURL(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("\t") // <- invalid URL
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected", err)
	}
}

func TestRunWithNoPort(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("stun://stun.ekiga.net")
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, errStunMissingPortInURL) {
		t.Fatal("not the error we expected", err)
	}
}

func TestRunWithInput(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget(defaultInput)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.Failure != nil {
		t.Fatal("expected nil failure here")
	}
	if tk.Endpoint != defaultEndpoint {
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
	measurer := NewExperimentMeasurer(Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget(defaultInput)
	err := measurer.Run(
		ctx,
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if *tk.Failure != "interrupted" {
		t.Fatal("expected different failure here")
	}
	if tk.Endpoint != defaultEndpoint {
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
	if _, ok := sk.(SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestNewClientFailure(t *testing.T) {
	config := &Config{}
	expected := errors.New("mocked error")
	config.newClient = func(conn stun.Connection, options ...stun.ClientOption) (*stun.Client, error) {
		return nil, expected
	}
	measurer := NewExperimentMeasurer(*config)
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget(defaultInput)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if !strings.HasPrefix(*tk.Failure, "unknown_failure") {
		t.Fatal("expected different failure here")
	}
	if tk.Endpoint != defaultEndpoint {
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
	config := &Config{}
	expected := errors.New("mocked error")
	config.dialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn := &FakeConn{WriteError: expected}
		return conn, nil
	}
	measurer := NewExperimentMeasurer(*config)
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget(defaultInput)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if !strings.HasPrefix(*tk.Failure, "unknown_failure") {
		t.Fatal("expected different failure here")
	}
	if tk.Endpoint != defaultEndpoint {
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
	config := &Config{}
	expected := errors.New("mocked error")
	config.dialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn := &FakeConn{ReadError: expected}
		return conn, nil
	}
	measurer := NewExperimentMeasurer(*config)
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget(defaultInput)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, stun.ErrTransactionTimeOut) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if *tk.Failure != netxlite.FailureGenericTimeoutError {
		t.Fatal("expected different failure here")
	}
	if tk.Endpoint != defaultEndpoint {
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
	measurement := &model.Measurement{TestKeys: &TestKeys{}}
	m := &Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(SummaryKeys)
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}
